package collection

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) intersection(ctx context.Context, job *base.Job) error {
	var inputStruct intersectionInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	if len(inputStruct.Data) == 0 {
		return job.Output.WriteData(ctx, intersectionOutput{Data: data.Array{}})
	}

	firstElem := inputStruct.Data[0]

	// Check if all elements are of the same type
	for i, val := range inputStruct.Data {
		if i > 0 && fmt.Sprintf("%T", val) != fmt.Sprintf("%T", firstElem) {
			return fmt.Errorf("all elements must be of the same type: expected %T, got %T at index %d", firstElem, val, i)
		}
	}

	switch firstElem.(type) {
	case data.Array:
		return handleArrayIntersection(ctx, job, inputStruct.Data)
	case data.Map:
		return handleObjectIntersection(ctx, job, inputStruct.Data)
	default:
		return fmt.Errorf("unsupported type for intersection: %T (must be either array or object)", firstElem)
	}
}

func handleArrayIntersection(ctx context.Context, job *base.Job, d []format.Value) error {
	if len(d) == 0 {
		return job.Output.WriteData(ctx, intersectionOutput{Data: data.Array{}})
	}

	firstArray := d[0].(data.Array)
	result := make([]format.Value, 0)

	for _, v := range firstArray {
		inAll := true
		for _, otherArray := range d[1:] {
			found := false
			for _, v2 := range otherArray.(data.Array) {
				if v.Equal(v2) {
					found = true
					break
				}
			}
			if !found {
				inAll = false
				break
			}
		}
		if inAll {
			isDuplicate := false
			for _, r := range result {
				if v.Equal(r) {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				result = append(result, v)
			}
		}
	}

	return job.Output.WriteData(ctx, intersectionOutput{Data: data.Array(result)})
}

func handleObjectIntersection(ctx context.Context, job *base.Job, d []format.Value) error {
	if len(d) == 0 {
		return job.Output.WriteData(ctx, intersectionOutput{Data: data.Map{}})
	}

	firstObj := d[0].(data.Map)
	result := make(map[string]format.Value)

	for key, value := range firstObj {
		inAll := true
		for _, otherObj := range d[1:] {
			if otherValue, exists := otherObj.(data.Map)[key]; !exists || !value.Equal(otherValue) {
				inAll = false
				break
			}
		}
		if inAll {
			result[key] = value
		}
	}

	return job.Output.WriteData(ctx, intersectionOutput{Data: data.Map(result)})
}
