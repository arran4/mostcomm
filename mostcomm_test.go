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

func TestDetectDuplicates_CollisionBug(t *testing.T) {
	// Bug: fp.Hash is not initialized with the first line's hash, but empty hash.
	// This causes ranges that differ only in the first line to have the same hash
	// if the subsequent lines are identical.

	// Case:
	// A: X Y
	// B: X Y
	// C: Z Y
	// D: Z Y

	// We expect:
	// 1 duplicate group for "X Y" (A and B).
	// 1 duplicate group for "Z Y" (C and D).
	// 1 duplicate group for "Y" (A, B, C, D) -- this is a suffix match which is also found.
	// Total 3 groups (of length >= 2, since Y\n is followed by empty line, it's 2 lines).

	// If bug exists:
	// "X Y" hash = MD5(H(Y)) (because X is ignored)
	// "Z Y" hash = MD5(H(Y)) (because Z is ignored)
	// They collide.
	// Result: 1 duplicate group containing A, B, C, D (merged).
	// Plus 1 duplicate group for "Y".
	// Total 2 groups.

	fsys := fstest.MapFS{
		"a.txt": {Data: []byte("X\nY\n")},
		"b.txt": {Data: []byte("X\nY\n")},
		"c.txt": {Data: []byte("Z\nY\n")},
		"d.txt": {Data: []byte("Z\nY\n")},
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

	duplicates := data.DetectDuplicates(func(fpm *mostcomm.FilePositionMatch) bool {
		return fpm.FilePosition.Lines() >= 2
	})

	if len(duplicates) != 3 {
		t.Errorf("Expected 3 duplicates (of length >= 2), got %d", len(duplicates))
		for _, d := range duplicates {
			t.Logf("Dup: %s", d)
		}
	}
}
