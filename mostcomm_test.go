package mostcomm_test

import (
	"io/fs"
	"mostcomm"
	"sync"
	"testing"
	"testing/fstest"
)

func TestDetectDuplicates_Integration(t *testing.T) {
	// Construct a virtual filesystem with test data
	// Intentionally omitting trailing newlines to match expected duplicate behavior
	fsys := fstest.MapFS{
		"a.txt": {Data: []byte("1\n2\n3\n4\n5\n6\n7\n8\n9\n10")},
		"b.txt": {Data: []byte("1\n2\n3\n4\n5\n\n\n6\n7\n8\n9\n10")},
		"c.txt": {Data: []byte("6\n7\n8\n9\n10\n6\n7\n8\n9\n10")},
	}

	data := &mostcomm.Data{
		Files:       map[string]*mostcomm.File{},
		Lines:       map[[16]byte][]*mostcomm.Line{},
		WalkerGroup: sync.WaitGroup{},
		FS:          fsys,
		LineMutex:   sync.Mutex{},
	}

	if err := fs.WalkDir(fsys, ".", mostcomm.Walker(data, []string{"*.txt"})); err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
	data.WalkerGroup.Wait()

	if len(data.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(data.Files))
	}

	duplicates := data.DetectDuplicates(func(fpm *mostcomm.FilePositionMatch) bool { return true })

	// Based on README output:
	// 2 Duplicate runs
	// - a.txt:0-4 (5), b.txt:0-4 (5)
	// - a.txt:5-9 (5), b.txt:7-11 (5), c.txt:0-4 (5), c.txt:5-9 (5)

	// We expect 2 duplicates.
	if len(duplicates) != 2 {
		t.Errorf("Expected 2 duplicates, got %d", len(duplicates))
		for _, d := range duplicates {
			t.Logf("Dup: %s", d)
		}
	}
}

func BenchmarkFilePositionString(b *testing.B) {
	fp := &mostcomm.FilePosition{
		Start: &mostcomm.Line{Position: 100},
		End:   &mostcomm.Line{Position: 200},
		File:  &mostcomm.File{Filename: "path/to/some/long/filename.go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fp.String()
	}
}
