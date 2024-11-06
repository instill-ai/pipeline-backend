package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) union(ctx context.Context, job *base.Job) error {
	in := &unionInput{}
	err := job.Input.ReadData(ctx, in)
	if err != nil {
		return err
	}
	sets := in.Sets

	u := make([]format.Value, 0)

	// Time complexity: O(n^2) where n is the total number of elements across
	// all sets. This is because for each element in the sets, we are
	// potentially iterating over the entire union set to check for equality.
	// Note: This time complexity can be improved to O(n) by implementing a hash
	// function for format.Value, allowing us to use a map for faster lookups.
	for _, s := range sets {
		for _, v := range s {
			found := false
			for _, uv := range u {
				if v.Equal(uv) {
					found = true
					break
				}
			}
			if !found {
				u = append(u, v)
			}
		}
	}

	return job.Output.WriteData(ctx, &unionOutput{Set: u})
}
