package collection

import (
	"context"
	"fmt"
	"sort"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) split(ctx context.Context, job *base.Job) error {
	var inputStruct splitInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	switch v := inputStruct.Data.(type) {
	case data.Array:
		if inputStruct.Size <= 0 {
			return fmt.Errorf("size must be greater than 0 for array splitting")
		}
		return handleArraySplit(ctx, job, v, inputStruct.Size)

	case data.Map:
		if inputStruct.Size <= 0 {
			return fmt.Errorf("size must be greater than 0 for object splitting")
		}
		return handleObjectSplit(ctx, job, v, inputStruct.Size)

	default:
		return fmt.Errorf("unsupported type for split: %T (must be array or object)", v)
	}
}

func handleArraySplit(ctx context.Context, job *base.Job, arr data.Array, size int) error {
	var chunks []format.Value

	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		chunk := data.Array(arr[i:end])
		chunks = append(chunks, chunk)
	}

	return job.Output.WriteData(ctx, splitOutput{Data: data.Array(chunks)})
}

func handleObjectSplit(ctx context.Context, job *base.Job, obj data.Map, size int) error {
	if size <= 0 {
		return fmt.Errorf("size must be greater than 0 for object splitting")
	}

	// Get all keys and sort them for consistent ordering
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var chunks []format.Value
	tempMap := make(data.Map)
	count := 0

	// Iterate through sorted keys
	for _, key := range keys {
		tempMap[key] = obj[key]
		count++

		if count == size {
			// Create a new map for this chunk
			newMap := make(data.Map, len(tempMap))
			for k, v := range tempMap {
				newMap[k] = v
			}
			chunks = append(chunks, newMap)
			tempMap = make(data.Map)
			count = 0
		}
	}

	// Add remaining items
	if len(tempMap) > 0 {
		newMap := make(data.Map, len(tempMap))
		for k, v := range tempMap {
			newMap[k] = v
		}
		chunks = append(chunks, newMap)
	}

	return job.Output.WriteData(ctx, splitOutput{Data: data.Array(chunks)})
}
