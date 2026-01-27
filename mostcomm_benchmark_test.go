package mostcomm_test

import (
	"fmt"
	"io/fs"
	"mostcomm"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func BenchmarkDetectDuplicates(b *testing.B) {
	// Generate some data
	lines := []string{}
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	content := strings.Join(lines, "\n")

	// Create multiple files with this content, some repeated
	fsys := fstest.MapFS{}
	// 20 files, 100 lines each.
    // Each line appears 20 times total.
    // For each line, we iterate 19 other occurrences.
    // 20 files * 100 lines * 19 matches = 38,000 iterations.
    // Maybe increase to get more work.
    // 50 files * 200 lines -> 50 * 200 * 49 = 490,000 iterations.
	for i := 0; i < 50; i++ {
		fsys[fmt.Sprintf("file%d.txt", i)] = &fstest.MapFile{
			Data: []byte(content), // All files are identical
		}
	}

	data := &mostcomm.Data{
		Files:       map[string]*mostcomm.File{},
		Lines:       map[[16]byte][]*mostcomm.Line{},
		WalkerGroup: sync.WaitGroup{},
		FS:          fsys,
		LineMutex:   sync.Mutex{},
	}

	if err := fs.WalkDir(fsys, ".", mostcomm.Walker(data, []string{"*.txt"})); err != nil {
		b.Fatalf("WalkDir failed: %v", err)
	}
	data.WalkerGroup.Wait()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        data.DetectDuplicates(func(fpm *mostcomm.FilePositionMatch) bool { return true })
    }
}
