package mostcomm

import "testing"

func TestDeleteMatchMax(t *testing.T) {
	f1 := &File{
		Count:    10,
		Filename: "f1",
	}
	fp1 := &FilePosition{
		File: f1,
	}
	fp3 := &FilePosition{
		File: f1,
	}
	f2 := &File{
		Count:    10,
		Filename: "f2",
	}
	fp2 := &FilePosition{
		File: f2,
	}
	dup1 := &Duplicate{
		FilePositions: []*FilePosition{
			fp1,
			fp2,
		},
	}
	dup2 := &Duplicate{
		FilePositions: []*FilePosition{
			fp3,
		},
	}
	tests := []struct {
		name       string
		duplicates []*Duplicate
		mm         int
		len        int
	}{
		{
			name:       "empty works",
			duplicates: []*Duplicate{},
			mm:         -1,
			len:        0,
		},
		{
			name: "Single element no removal disabled",
			duplicates: []*Duplicate{
				dup1,
			},
			mm:  -1,
			len: 1,
		},
		{
			name: "Single element no removal high threshold",
			duplicates: []*Duplicate{
				dup1,
			},
			mm:  4,
			len: 1,
		},
		{
			name: "Single element one removal",
			duplicates: []*Duplicate{
				dup1,
			},
			mm:  1,
			len: 0,
		},
		{
			name: "Remove first dup, not second",
			duplicates: []*Duplicate{
				dup1,
				dup2,
			},
			mm:  1,
			len: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dups := DeleteMatchMax(tt.duplicates, tt.mm)
			if tt.len != len(dups) {
				t.Errorf("Expected len %d but got %d", tt.len, len(dups))
			}
		})
	}
}
