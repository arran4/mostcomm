package mostcomm

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
)

// Run is a subcommand `mostcomm`
//
// Detects duplicates in files based on common lines.
//
// Flags:
//
//	dir:               --dir               (default: ".")         Directory to scan
//	fileMask:          --mask              (default: "*.txt")     File glob mask to scan. , separated
//	sort:              --sort              (default: "none")      Sorting order, algorithms; none, lines, average-coverage
//	sortDirect:        --sort-direction    (default: "ascending") Sorting direction, algorithms; ascending, descending
//	thresholdPercent:  --percent-threshold (default: 0)           Minimum required % of the file in common
//	thresholdLines:    --lines-threshold   (default: 0)           Minimum required lines of the file in common
//	thresholdMatchMax: --match-max-threshold (default: -1)        Maximum time a match is allowed
//	concurrency:       --concurrency       (default: 0)           Concurrency limit (default 0 = number of CPUs)
func Run(dir, fileMask, sort, sortDirect string, thresholdPercent, thresholdLines, thresholdMatchMax, concurrency int) {
	data := &Data{
		Files:       map[string]*File{},
		Lines:       map[[16]byte][]*Line{},
		WalkerGroup: sync.WaitGroup{},
		FS:          os.DirFS(dir),
		LineMutex:   sync.Mutex{},
		Concurrency: concurrency,
	}
	if err := fs.WalkDir(data.FS, ".", Walker(data, strings.Split(fileMask, ";"))); err != nil {
		log.Panic(err)
	}
	data.WalkerGroup.Wait()
	_ = os.Stderr.Sync()
	fmt.Printf("%d files scanned\n", len(data.Files))
	fmt.Printf("%d unique lines scanned\n", len(data.Lines))
	fmt.Printf("%d total lines scanned\n", data.TotalLines())
	duplicates := data.DetectDuplicates(thresholdFunc(thresholdPercent, thresholdLines))
	fmt.Printf("%d Duplicate founds\n", len(duplicates))
	if thresholdMatchMax >= 0 {
		duplicates = DeleteMatchMax(duplicates, thresholdMatchMax)
	}
	compare := func(a, b int) int { return a - b }
	switch sortDirect {
	case "ascending":

	case "descending":
		compare = func(a, b int) int {
			return b - a
		}
	}
	switch sort {
	case "none":
	case "lines":
		slices.SortFunc(duplicates, func(a, b *Duplicate) int {
			return compare(a.TotalLines(), b.TotalLines())
		})
	case "average-coverage":
		slices.SortFunc(duplicates, func(a, b *Duplicate) int {
			return compare(a.AverageCoveragePercent(), b.AverageCoveragePercent())
		})
	}
	for _, dup := range duplicates {
		fmt.Printf("- %s\n", dup)
	}
}

func thresholdFunc(thresholdLines int, thresholdPercent int) func(fpm *FilePositionMatch) bool {
	return func(fpm *FilePositionMatch) bool {
		if thresholdLines > 0 && thresholdLines > fpm.FilePosition.Lines() {
			return false
		}
		if thresholdPercent > 0 && thresholdPercent > fpm.FilePosition.Percent() {
			return false
		}
		return true
	}
}
