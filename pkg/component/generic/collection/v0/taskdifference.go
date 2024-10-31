package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) difference(ctx context.Context, job *base.Job) error {
	in := &differenceInput{}
	if err := job.Input.ReadData(ctx, in); err != nil {
		return err
	}

	setA := in.SetA
	setB := in.SetB

	set := make([]format.Value, 0, len(setA))

	// Time complexity: O(n*m) where n is the size of setA and m is the size of
	// setB. This is because for each element in setA, we are potentially
	// iterating over all elements in setB to check for equality.
	// Note: We can provide a better time complexity after we implement a hash
	// function for format.Value, allowing us to use a map for faster lookups.
	for _, v := range setA {
		found := false
		for _, b := range setB {
			if v.Equal(b) {
				found = true
				break
			}
		}
		if !found {
			set = append(set, v)
		}
	}

	differenceOutput := &differenceOutput{Set: set}
	return job.Output.WriteData(ctx, differenceOutput)
}
