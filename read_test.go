package mostcomm_test

import (
	"mostcomm"
	"sync"
	"testing"
	"testing/fstest"
)

func TestReadBehavior(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int // number of lines
	}{
		{"NoNewline", "abc", 1},
		{"WithNewline", "abc\n", 2},
		{"TwoNewlines", "abc\n\n", 3},
		{"Empty", "", 1}, // Loop runs for i=0 (len=0), r='\n', creates one empty line?
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"test.txt": {Data: []byte(tt.content)},
			}
			data := &mostcomm.Data{
				Files:       map[string]*mostcomm.File{},
				Lines:       map[[16]byte][]*mostcomm.Line{},
				WalkerGroup: sync.WaitGroup{},
				FS:          fsys,
				LineMutex:   sync.Mutex{},
			}

			f := &mostcomm.File{
				Data:     data,
				Filename: "test.txt",
			}
			data.Files["test.txt"] = f
			data.WalkerGroup.Add(1)

			c := make(chan struct{}, 1)
			// Read is called synchronously here.
			// It will push to c, do work, and pop from c in defer.
			// Since c has buffer 1, it won't block.
			f.Read(c)

			// Count lines in the file
			count := 0
			curr := f.Head
			for curr != nil {
				count++
				// t.Logf("Line %d hash: %x", count, curr.Hash)
				curr = curr.Next
			}

			if count != tt.expected {
				t.Errorf("Expected %d lines, got %d", tt.expected, count)
			}

            // Also verify hashes if needed, but count is a good start
		})
	}
}
