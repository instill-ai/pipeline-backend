package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) intersection(ctx context.Context, job *base.Job) error {
	in := &intersectionInput{}
	err := job.Input.ReadData(ctx, in)
	if err != nil {
		return err
	}
	sets := in.Sets

	if len(sets) == 0 {
		return job.Output.WriteData(ctx, &intersectionOutput{Set: []format.Value{}})
	}

	set := make([]format.Value, 0, len(sets[0]))
	set = append(set, sets[0]...)

	// Time complexity: O(n^2) where n is the size of the largest set. This is
	// because for each element in the current set, we are iterating over the
	// entire set to find matches.
	// Note: This time complexity can be improved to O(n) by implementing a hash
	// function for format.Value and using a map to store the set elements.
	for _, s := range sets[1:] {
		tempSet := make([]format.Value, 0, len(s))
		for _, v := range s {
			for _, v2 := range set {
				if v.Equal(v2) {
					tempSet = append(tempSet, v)
					break
				}
			}
		}
		set = tempSet
	}

	// Remove duplicates from the result set
	uniqueSet := make([]format.Value, 0, len(set))
	for _, v := range set {
		isDuplicate := false
		for _, u := range uniqueSet {
			if v.Equal(u) {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			uniqueSet = append(uniqueSet, v)
		}
	}

	set = uniqueSet

	return job.Output.WriteData(ctx, &intersectionOutput{Set: set})
}
