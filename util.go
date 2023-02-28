package mostcomm

import "golang.org/x/exp/slices"

func DeleteMatchMax(duplicates []*Duplicate, mm int) []*Duplicate {
	rmFrom, i := -1, 0
	for i < len(duplicates) {
		dup := duplicates[i]
		del := mm > 0 && len(dup.Files()) > mm
		if del {
			if rmFrom == -1 {
				rmFrom = i
			}
		} else if rmFrom != -1 {
			duplicates = slices.Delete(duplicates, rmFrom, i)
			i, rmFrom = rmFrom, -1
			continue
		}
		i++
	}
	if rmFrom != -1 {
		duplicates = slices.Delete(duplicates, rmFrom, i)
	}
	return duplicates
}
