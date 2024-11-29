package collection

import (
	"context"
	"fmt"
	"sort"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) symmetricDifference(ctx context.Context, job *base.Job) error {
	var inputStruct symmetricDifferenceInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	if len(inputStruct.Data) == 0 {
		return job.Output.WriteData(ctx, symmetricDifferenceOutput{Data: data.Array{}})
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
		return handleArraySymmetricDifference(ctx, job, inputStruct.Data)
	case data.Map:
		return handleObjectSymmetricDifference(ctx, job, inputStruct.Data)
	default:
		return fmt.Errorf("unsupported type for symmetric difference: %T (must be either array or object)", firstElem)
	}
}

func handleArraySymmetricDifference(ctx context.Context, job *base.Job, d data.Array) error {
	if len(d) < 2 {
		return job.Output.WriteData(ctx, symmetricDifferenceOutput{Data: d[0]})
	}

	result := make(map[string]format.Value)
	counts := make(map[string]int)

	// Count occurrences of each value
	for _, arr := range d {
		seen := make(map[string]bool)
		for _, v := range arr.(data.Array) {
			key := v.String()
			if !seen[key] {
				counts[key]++
				result[key] = v
				seen[key] = true
			}
		}
	}

	// Keep only elements that appear exactly once
	symmetricDiff := make([]format.Value, 0)
	for key, count := range counts {
		if count == 1 {
			symmetricDiff = append(symmetricDiff, result[key])
		}
	}

	sortedArray := sortArray(data.Array(symmetricDiff))
	outputStruct := symmetricDifferenceOutput{Data: sortedArray}
	return job.Output.WriteData(ctx, outputStruct)
}

func handleObjectSymmetricDifference(ctx context.Context, job *base.Job, d data.Array) error {
	if len(d) < 2 {
		return job.Output.WriteData(ctx, symmetricDifferenceOutput{Data: d[0]})
	}

	result := make(map[string]format.Value)
	counts := make(map[string]int)

	// Count occurrences of each key-value pair
	for _, obj := range d {
		seen := make(map[string]bool)
		for key, value := range obj.(data.Map) {
			mapKey := key + ":" + value.String()
			if !seen[mapKey] {
				counts[mapKey]++
				result[mapKey] = value
				seen[mapKey] = true
			}
		}
	}

	// Keep only key-value pairs that appear exactly once
	symmetricDiff := make(map[string]format.Value)
	for mapKey, count := range counts {
		if count == 1 {
			key := mapKey[:len(mapKey)-len(result[mapKey].String())-1]
			symmetricDiff[key] = result[mapKey]
		}
	}

	outputStruct := symmetricDifferenceOutput{Data: data.Map(symmetricDiff)}
	return job.Output.WriteData(ctx, outputStruct)
}

func sortArray(arr data.Array) data.Array {
	sort.Slice(arr, func(i, j int) bool {
		// Convert values to strings for consistent comparison
		return fmt.Sprint(arr[i]) < fmt.Sprint(arr[j])
	})
	return arr
}
