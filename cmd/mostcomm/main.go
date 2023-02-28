package main

import (
	"flag"
	"fmt"
	"golang.org/x/exp/slices"
	"io/fs"
	"log"
	"mostcomm"
	"os"
	"strings"
	"sync"
)

var (
	dirFlag               = flag.String("dir", ".", "Directory to scan")
	fileMaskFlag          = flag.String("mask", "*.txt", "File glob mask to scan. , separated")
	sortFlag              = flag.String("sort", "none", "Sorting order, algorithms; none, lines, average-coverage")
	sortDirectFlag        = flag.String("sort-direction", "ascending", "Sorting direction, algorithms; ascending, descending")
	thresholdPercentFlag  = flag.Int("percent-threshold", 0, "Minimum required % of the file in common")
	thresholdLinesFlag    = flag.Int("lines-threshold", 0, "Minimum required lines of the file in common")
	thresholdMatchMaxFlag = flag.Int("match-max-threshold", -1, "Maximum time a match is allowed")
)

func main() {
	flag.Parse()
	data := &mostcomm.Data{
		Files:       map[string]*mostcomm.File{},
		Lines:       map[[16]byte][]*mostcomm.Line{},
		WalkerGroup: sync.WaitGroup{},
		FS:          os.DirFS(*dirFlag),
		LineMutex:   sync.Mutex{},
	}
	if err := fs.WalkDir(data.FS, ".", mostcomm.Walker(data, strings.Split(*fileMaskFlag, ";"))); err != nil {
		log.Panic(err)
	}
	data.WalkerGroup.Wait()
	_ = os.Stderr.Sync()
	fmt.Printf("%d files scanned\n", len(data.Files))
	fmt.Printf("%d unique lines scanned\n", len(data.Lines))
	fmt.Printf("%d total lines scanned\n", data.TotalLines())
	duplicates := data.DetectDuplicates(thresholdFunc(*thresholdPercentFlag, *thresholdLinesFlag))
	fmt.Printf("%d Duplicate founds\n", len(duplicates))
	if *thresholdMatchMaxFlag >= 0 {
		duplicates = mostcomm.DeleteMatchMax(duplicates, *thresholdMatchMaxFlag)
	}
	direction := func(b bool) bool { return b }
	switch *sortDirectFlag {
	case "ascending":

	case "descending":
		direction = func(b bool) bool {
			return !b
		}
	}
	switch *sortFlag {
	case "none":
	case "lines":
		slices.SortFunc(duplicates, func(a, b *mostcomm.Duplicate) bool {
			return direction(a.TotalLines() < b.TotalLines())
		})
	case "average-coverage":
		slices.SortFunc(duplicates, func(a, b *mostcomm.Duplicate) bool {
			return direction(a.AverageCoveragePercent() < b.AverageCoveragePercent())
		})
	}
	for _, dup := range duplicates {
		fmt.Printf("- %s\n", dup)
	}
}

func thresholdFunc(thresholdLines int, thresholdPercent int) func(fpm *mostcomm.FilePositionMatch) bool {
	return func(fpm *mostcomm.FilePositionMatch) bool {
		if thresholdLines > 0 && thresholdLines > fpm.FilePosition.Lines() {
			return false
		}
		if thresholdPercent > 0 && thresholdPercent > fpm.FilePosition.Percent() {
			return false
		}
		return true
	}
}
