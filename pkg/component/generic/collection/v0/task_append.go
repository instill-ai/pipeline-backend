package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) append(ctx context.Context, job *base.Job) error {
	var inputStruct appendInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	var result format.Value
	switch v := inputStruct.Data.(type) {
	case data.Array:
		// Already an array, just append the value
		result = append(v, toArrayElement(inputStruct.Value))
	case data.Map:
		// If value is a map, merge its key-value pairs
		if valueMap, ok := inputStruct.Value.(data.Map); ok {
			for k, val := range valueMap {
				v[k] = val
			}
			result = v
		} else {
			// Convert map to array element and append new value
			arr := []format.Value{toArrayElement(v)}
			result = data.Array(append(arr, toArrayElement(inputStruct.Value)))
		}
	default:
		// Convert primitive to array and append
		arr := []format.Value{toArrayElement(inputStruct.Data)}
		result = data.Array(append(arr, toArrayElement(inputStruct.Value)))
	}

	outputStruct := appendOutput{Data: result}
	return job.Output.WriteData(ctx, outputStruct)
}

func toArrayElement(v interface{}) format.Value {
	switch val := v.(type) {
	case format.Value:
		return val
	case string:
		return data.NewString(val)
	case int:
		return data.NewNumberFromInteger(val)
	case float64:
		return data.NewNumberFromFloat(val)
	case bool:
		return data.NewBoolean(val)
	default:
		return v.(format.Value)
	}
}
