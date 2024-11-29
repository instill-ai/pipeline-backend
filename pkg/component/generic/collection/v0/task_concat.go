package collection

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) concat(ctx context.Context, job *base.Job) error {
	var inputStruct concatInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	if len(inputStruct.Data) == 0 {
		return job.Output.WriteData(ctx, concatOutput{Data: data.Array{}})
	}

	// Determine the type based on the first element
	firstElem := inputStruct.Data[0]

	// First check if all elements are of the same type
	for i, val := range inputStruct.Data {
		if i > 0 && fmt.Sprintf("%T", val) != fmt.Sprintf("%T", firstElem) {
			return fmt.Errorf("all elements must be of the same type: expected %T, got %T at index %d", firstElem, val, i)
		}
	}

	switch firstElem.(type) {
	case data.Array:
		// Handle array concatenation
		result := []format.Value{}
		for _, val := range inputStruct.Data {
			arr, ok := val.(data.Array)
			if !ok {
				return fmt.Errorf("expected array, got %T", val)
			}
			result = append(result, arr...)
		}
		return job.Output.WriteData(ctx, concatOutput{Data: data.Array(result)})

	case data.Map:
		// Handle object merging
		mergedMap := data.Map{}
		for _, val := range inputStruct.Data {
			obj, ok := val.(data.Map)
			if !ok {
				return fmt.Errorf("expected object, got %T", val)
			}
			// Merge objects (later values override earlier ones)
			for k, v := range obj {
				mergedMap[k] = v
			}
		}
		// Wrap the merged map in an array with single element
		return job.Output.WriteData(ctx, concatOutput{Data: mergedMap})

	default:
		return fmt.Errorf("unsupported type for concatenation: %T (must be either array or object)", firstElem)
	}
}
