package collection

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) union(ctx context.Context, job *base.Job) error {
	var inputStruct unionInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	if len(inputStruct.Data) == 0 {
		return job.Output.WriteData(ctx, unionOutput{Data: data.Array{}})
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
		return handleArrayUnion(ctx, job, inputStruct.Data)
	case data.Map:
		return handleObjectUnion(ctx, job, inputStruct.Data)
	default:
		return fmt.Errorf("unsupported type for union: %T (must be either array or object)", firstElem)
	}
}

func handleArrayUnion(ctx context.Context, job *base.Job, d []format.Value) error {

	result := make([]format.Value, 0)

	for _, arr := range d {
		for _, v := range arr.(data.Array) {
			found := false
			for _, r := range result {
				if v.Equal(r) {
					found = true
					break
				}
			}
			if !found {
				result = append(result, v)
			}
		}
	}

	return job.Output.WriteData(ctx, unionOutput{Data: data.Array(result)})
}

func handleObjectUnion(ctx context.Context, job *base.Job, d []format.Value) error {

	result := make(map[string]format.Value)

	for _, obj := range d {
		for key, value := range obj.(data.Map) {
			result[key] = value
		}
	}

	return job.Output.WriteData(ctx, unionOutput{Data: data.Map(result)})
}
