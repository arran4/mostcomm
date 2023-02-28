package mostcomm

import (
	"crypto/md5"
	"fmt"
	"golang.org/x/exp/maps"
	"hash"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type File struct {
	Head     *Line
	Tail     *Line
	Count    int
	Data     *Data
	Filename string
}

func (f *File) Read(c chan struct{}) {
	c <- struct{}{}
	defer func() {
		f.Data.WalkerGroup.Done()
		<-c
	}()
	b, err := fs.ReadFile(f.Data.FS, f.Filename)
	if err != nil {
		log.Panic(err)
	}
	lines := make([]*Line, 0, 1024)
	var prev *Line
	for i, first, last := 0, 0, 0; i <= len(b); i++ {
		var r byte = '\n'
		if i < len(b) {
			r = b[i]
		}
		switch r {
		case '\r':
		case '\n':
			l := &Line{
				File:     f,
				Prev:     prev,
				Position: f.Count,
				Hash:     md5.Sum(b[first : last+1]),
			}
			f.Count++
			if prev != nil {
				prev.Next = l
			} else {
				f.Head = l
			}
			lines = append(lines, l)
			if len(lines) >= 1024 {
				f.Data.Add(lines)
				lines = lines[:0]
			}
			prev = l
			first = i + 1
			last = i
		default:
			last = i
		}
	}
	f.Tail = prev
	f.Data.Add(lines)
}

type Line struct {
	File     *File
	Prev     *Line
	Next     *Line
	Position int
	Hash     [16]byte
}

type Data struct {
	Files       map[string]*File
	Lines       map[[16]byte][]*Line
	WalkerGroup sync.WaitGroup
	FS          fs.FS
	LineMutex   sync.Mutex
}

func (d *Data) Add(lines []*Line) {
	d.LineMutex.Lock()
	defer d.LineMutex.Unlock()
	for _, l := range lines {
		d.Lines[l.Hash] = append(d.Lines[l.Hash], l)
	}
}

func (d *Data) TotalLines() int {
	r := 0
	for _, f := range d.Files {
		r += f.Count
	}
	return r
}

type FilePosition struct {
	Start, End *Line
	File       *File
}

func (fp *FilePosition) String() string {
	return fmt.Sprintf("%s:%d-%d (%d)", fp.File.Filename, fp.Start.Position, fp.End.Position, fp.Lines())
}

func (fpm *FilePositionMatch) HashKey() (b [16]byte) {
	copy(b[:], fpm.Hash.Sum(nil))
	return
}

func (fp *FilePosition) Duplicate() *Duplicate {
	return &Duplicate{
		FilePositions: []*FilePosition{fp},
		Head:          fp.Start.Hash,
		Tail:          fp.End.Hash,
	}
}

func (fp *FilePosition) Postions() [2]int {
	return [2]int{fp.Start.Position, fp.End.Position}
}

func (fp *FilePosition) Lines() int {
	return fp.End.Position - fp.Start.Position + 1
}

func (fp *FilePosition) Percent() (r int) {
	r += fp.Lines() * 10000 / fp.File.Count
	r /= 100
	return
}

type Duplicate struct {
	FilePositions []*FilePosition
	Head, Tail    [16]byte
}

func (d *Duplicate) String() string {
	var ss []string
	for _, fp := range d.FilePositions {
		ss = append(ss, fp.String())
	}
	sort.Strings(ss)
	return strings.Join(ss, ", ")
}

func (d *Duplicate) TotalLines() (r int) {
	for _, fp := range d.FilePositions {
		r += fp.Lines()
	}
	return
}

func (d *Duplicate) AverageCoveragePercent() (r int) {
	type T struct {
		Lines int
		Total int
	}
	fm := map[*File]T{}
	for _, fp := range d.FilePositions {
		f := fm[fp.File]
		f.Lines += fp.Lines()
		f.Total = fp.File.Count
		fm[fp.File] = f
	}
	for _, f := range fm {
		r += f.Lines * 10000 / f.Total
	}
	r /= 100 * len(fm)
	return
}

func (d *Duplicate) Files() []*File {
	files := map[*File]struct{}{}
	for _, fp := range d.FilePositions {
		files[fp.File] = struct{}{}
	}
	return maps.Keys(files)
}

type FilePositionMatch struct {
	FilePosition *FilePosition
	With         *Line
	Hash         hash.Hash
}

func (d *Data) DetectDuplicates(keepFilter func(fpm *FilePositionMatch) bool) []*Duplicate {
	var dups []*Duplicate
	ranges := map[[16]byte]*Duplicate{}
	for _, f := range d.Files {
		var matches []*FilePositionMatch
		var seenPos = map[[2]int]struct{}{}
		for p := f.Head; p != nil; p = p.Next {
			var nextMatches []*FilePositionMatch
			var missedMatches []*FilePositionMatch
			var missedLines = map[*Line]*FilePositionMatch{}
			for _, l := range d.Lines[p.Hash] {
				if l.File == p.File {
					continue
				}
				fp := &FilePositionMatch{
					FilePosition: &FilePosition{
						Start: p,
						End:   p,
						File:  f,
					},
					Hash: md5.New(),
					With: l,
				}
				fp.Hash.Sum(p.Hash[:])
				missedLines[l] = fp
			}
			for _, fp := range matches {
				if fp.With != nil && fp.With.Next != nil {
					delete(missedLines, fp.With.Next)
					if fp.With.Next.Hash == p.Hash {
						fp.Next(p)
						nextMatches = append(nextMatches, fp)
						continue
					}
				}
				_, ok := seenPos[fp.Positions()]
				if ok {
					continue
				}
				seenPos[fp.Positions()] = struct{}{}
				missedMatches = append(missedMatches, fp)
			}
			for _, ml := range missedLines {
				nextMatches = append(nextMatches, ml)
			}
			for _, fp := range missedMatches {
				d, ok := ranges[fp.HashKey()]
				if ok {
					d.FilePositions = append(d.FilePositions, fp.FilePosition)
					continue
				}
				if !keepFilter(fp) {
					continue
				}
				d = fp.FilePosition.Duplicate()
				dups = append(dups, d)
				ranges[fp.HashKey()] = d
			}
			matches = nextMatches
		}
		for _, fp := range matches {
			_, ok := seenPos[fp.Positions()]
			if ok {
				continue
			}
			seenPos[fp.Positions()] = struct{}{}
			d, ok := ranges[fp.HashKey()]
			if ok {
				d.FilePositions = append(d.FilePositions, fp.FilePosition)
				continue
			}
			if !keepFilter(fp) {
				continue
			}
			d = fp.FilePosition.Duplicate()
			dups = append(dups, d)
			ranges[fp.HashKey()] = d
		}
	}
	return dups
}

func (fpm *FilePositionMatch) Next(p *Line) {
	fpm.FilePosition.End = p
	fpm.With = fpm.With.Next
	fpm.Hash.Write(p.Hash[:])
}

func (fpm *FilePositionMatch) Positions() [2]int {
	return fpm.FilePosition.Postions()
}

var _ fmt.Stringer = (*Duplicate)(nil)

func Walker(data *Data, masks []string) fs.WalkDirFunc {
	c := make(chan struct{}, 25)
	return func(path string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() {
			return nil
		}
		nomatch := true
		_, fn := filepath.Split(path)
		for _, mask := range masks {
			if m, err := filepath.Match(mask, fn); m {
				nomatch = false
			} else if err != nil {
				return fmt.Errorf("file match %q: %w", mask, err)
			}
		}
		if nomatch {
			return nil
		}
		f := &File{
			Data:     data,
			Head:     nil,
			Tail:     nil,
			Count:    0,
			Filename: path,
		}
		log.Printf("Scanning %s", path)
		data.Files[path] = f
		data.WalkerGroup.Add(1)
		go f.Read(c)
		return nil
	}
}
