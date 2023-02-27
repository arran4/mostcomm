package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"hash"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	dirFlag      = flag.String("dir", ".", "Directory to scan")
	fileMaskFlag = flag.String("mask", "*.txt", "File glob mask to scan. ; separated")
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

func (p *FilePosition) String() string {
	return fmt.Sprintf("%s:%d-%d (%d)", p.File.Filename, p.Start.Position, p.End.Position, p.End.Position-p.Start.Position+1)
}

func (p *FilePositionMatch) Key() (b [16]byte) {
	h := p.Hash.Sum(nil)
	for i, c := range h {
		b[i] = c
	}
	return
}

func (p *FilePosition) Duplicate() *Duplicate {
	return &Duplicate{
		Files: []*FilePosition{p},
		Head:  p.Start.Hash,
		Tail:  p.End.Hash,
	}
}

type Duplicate struct {
	Files      []*FilePosition
	Head, Tail [16]byte
}

func (d *Duplicate) String() string {
	var ss []string
	for _, fp := range d.Files {
		ss = append(ss, fp.String())
	}
	sort.Strings(ss)
	return strings.Join(ss, ", ")
}

type FilePositionMatch struct {
	FilePosition *FilePosition
	With         *Line
	Hash         hash.Hash
}

func (d *Data) DetectDuplicates() []*Duplicate {
	var dups []*Duplicate
	ranges := map[[16]byte]*Duplicate{}
	for _, f := range d.Files {
		var matches []*FilePositionMatch
		for p := f.Head; p != nil; p = p.Next {
			var nextMatches []*FilePositionMatch
			var missedMatches = map[*File]*FilePositionMatch{}
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
				missedMatches[fp.FilePosition.File] = fp
			}
			for _, ml := range missedLines {
				nextMatches = append(nextMatches, ml)
			}
			for _, fp := range missedMatches {
				d, ok := ranges[fp.Key()]
				if ok {
					d.Files = append(d.Files, fp.FilePosition)
					continue
				}
				d = fp.FilePosition.Duplicate()
				dups = append(dups, d)
				ranges[fp.Key()] = d
			}
			matches = nextMatches
		}
		for _, fp := range matches {
			d, ok := ranges[fp.Key()]
			if ok {
				d.Files = append(d.Files, fp.FilePosition)
				continue
			}
			d = fp.FilePosition.Duplicate()
			dups = append(dups, d)
			ranges[fp.Key()] = d
		}
	}
	return dups
}

func (fp *FilePositionMatch) Next(p *Line) {
	fp.FilePosition.End = p
	fp.With = fp.With.Next
	fp.Hash.Write(p.Hash[:])
}

func main() {
	flag.Parse()
	data := &Data{
		Files:       map[string]*File{},
		Lines:       map[[16]byte][]*Line{},
		WalkerGroup: sync.WaitGroup{},
		FS:          os.DirFS(*dirFlag),
		LineMutex:   sync.Mutex{},
	}
	if err := fs.WalkDir(data.FS, ".", Walker(data, strings.Split(*fileMaskFlag, ";"))); err != nil {
		log.Panic(err)
	}
	data.WalkerGroup.Wait()
	_ = os.Stderr.Sync()
	fmt.Printf("%d files scanned\n", len(data.Files))
	fmt.Printf("%d unique lines scanned\n", len(data.Lines))
	fmt.Printf("%d total lines scanned\n", data.TotalLines())
	duplicates := data.DetectDuplicates()
	fmt.Printf("%d Duplicate runs\n", len(duplicates))
	for _, dup := range duplicates {
		fmt.Printf("- %s\n", dup)
	}
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
