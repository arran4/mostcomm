package main

import (
	"crypto/md5"
	"flag"
	"fmt"
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
				Hash:     md5.Sum(b[first:last]),
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
			first = i
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
	return fmt.Sprintf("%s:%d-%d (%d)", p.File.Filename, p.Start.Position, p.End.Position, p.End.Position-p.Start.Position)
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

func (d *Data) DetectDuplicates() []*Duplicate {
	var dups []*Duplicate
	ranges := map[[2][16]byte]*Duplicate{}
	for _, f := range d.Files {
		matches := map[*File]*Line{}
		for p := f.Head; p != nil; p = p.Next {
			seen := map[*File]struct{}{}
			for _, l := range d.Lines[p.Hash] {
				seen[l.File] = struct{}{}
				if l == p {
					continue
				}
				_, ok := matches[l.File]
				if !ok {
					matches[l.File] = p
					continue
				}
			}
			var dff []*File
			for ff, sl := range matches {
				if ff == f || sl == p {
					continue
				}
				if _, ok := seen[ff]; ok {
					continue
				}
				fp := &FilePosition{
					Start: sl,
					End:   p,
					File:  f,
				}
				k := [2][16]byte{sl.Hash, p.Hash}
				d, ok := ranges[k]
				if ok {
					d.Files = append(d.Files, fp)
					continue
				}
				d = &Duplicate{
					Files: []*FilePosition{fp},
					Head:  sl.Hash,
					Tail:  p.Hash,
				}
				dups = append(dups, d)
				ranges[k] = d
				dff = append(dff, ff)
			}
			for _, ff := range dff {
				delete(matches, ff)
			}
		}
		for ff, sl := range matches {
			if ff == f || sl == f.Tail {
				continue
			}
			fp := &FilePosition{
				Start: sl,
				End:   f.Tail,
				File:  f,
			}
			k := [2][16]byte{sl.Hash, f.Tail.Hash}
			d, ok := ranges[k]
			if ok {
				d.Files = append(d.Files, fp)
				continue
			}
			d = &Duplicate{
				Files: []*FilePosition{fp},
				Head:  sl.Hash,
				Tail:  f.Tail.Hash,
			}
			dups = append(dups, d)
			ranges[k] = d
		}
	}
	return dups
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
