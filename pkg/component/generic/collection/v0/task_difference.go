package collection

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) difference(ctx context.Context, job *base.Job) error {
	var inputStruct differenceInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	if len(inputStruct.Data) == 0 {
		return job.Output.WriteData(ctx, differenceOutput{Data: data.Array{}})
	}

	firstElem := inputStruct.Data[0]

	// First check if all elements are of the same type
	for i, val := range inputStruct.Data {
		if i > 0 && fmt.Sprintf("%T", val) != fmt.Sprintf("%T", firstElem) {
			return fmt.Errorf("all elements must be of the same type: expected %T, got %T at index %d", firstElem, val, i)
		}
	}

	switch firstElem.(type) {
	case data.Array:
		return handleArrayDifference(ctx, job, inputStruct.Data)
	case data.Map:
		return handleObjectDifference(ctx, job, inputStruct.Data)
	default:
		return fmt.Errorf("unsupported type for concatenation: %T (must be either array or object)", firstElem)
	}
}

func handleArrayDifference(ctx context.Context, job *base.Job, d data.Array) error {
	setA := d[0].(data.Array)
	otherSets := d[1:]

	set := make([]format.Value, 0, len(setA))

	for _, v := range setA {
		found := false
		for _, otherSet := range otherSets {
			for _, b := range otherSet.(data.Array) {
				if v.Equal(b) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			set = append(set, v)
		}
	}

	outputStruct := differenceOutput{Data: data.Array(set)}
	return job.Output.WriteData(ctx, outputStruct)
}

func handleObjectDifference(ctx context.Context, job *base.Job, d data.Array) error {
	setA := d[0]
	otherSets := d[1:]

	set := make(map[string]format.Value)

	for key, value := range setA.(data.Map) {
		found := false
		for _, otherSet := range otherSets {
			if otherValue, exists := otherSet.(data.Map)[key]; exists && value.Equal(otherValue) {
				found = true
				break
			}
		}
		if !found {
			set[key] = value
		}
	}

	outputStruct := differenceOutput{Data: data.Map(set)}
	return job.Output.WriteData(ctx, outputStruct)
}
