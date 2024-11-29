package collection

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) assign(ctx context.Context, job *base.Job) error {
	var inputStruct assignInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	result, err := assignValueAtPath(inputStruct.Data, inputStruct.Path, inputStruct.Value)
	if err != nil {
		return err
	}

	out := assignOutput{Data: result}
	return job.Output.WriteData(ctx, out)
}

// assignValueAtPath handles nested path assignment like ".[0].key" or "users.[0].name"
func assignValueAtPath(curData format.Value, path string, value format.Value) (format.Value, error) {
	if path == "" {
		return value, nil
	}

	// Initialize root if it's nil
	if curData == nil {
		curData = data.Map{}
	}
	root := curData

	// Split path into segments
	segments := strings.Split(path, ".")
	current := curData
	var parent format.Value
	lastIdx := len(segments) - 1

	for i, segment := range segments {
		if segment == "" {
			continue
		}

		// Handle array index notation [n]
		if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
			indexStr := segment[1 : len(segment)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", indexStr)
			}

			// Check for negative index
			if index < 0 {
				return nil, fmt.Errorf("negative array index: %d", index)
			}

			// If current is nil or not an array, initialize it
			if current == nil {
				current = data.Array{}
			}
			arr, ok := current.(data.Array)
			if !ok {
				arr = data.Array{}
				if parent != nil {
					switch p := parent.(type) {
					case data.Map:
						p[segments[i-1]] = arr
					case data.Array:
						parentIdx, _ := strconv.Atoi(segments[i-1][1 : len(segments[i-1])-1])
						p[parentIdx] = arr
					}
				} else {
					root = arr
				}
			}

			// Check if index is too large
			if index > len(arr) {
				return nil, fmt.Errorf("array index out of bounds: %d (array length: %d)", index, len(arr))
			}

			// Extend array if needed
			if index >= len(arr) {
				arr = append(arr, data.Map{})
				if parent != nil {
					switch p := parent.(type) {
					case data.Map:
						p[segments[i-1]] = arr
					case data.Array:
						parentIdx, _ := strconv.Atoi(segments[i-1][1 : len(segments[i-1])-1])
						p[parentIdx] = arr
					}
				} else {
					root = arr
				}
			}

			if i == lastIdx {
				arr[index] = value
				return root, nil
			}

			if arr[index] == nil {
				arr[index] = data.Map{}
			}
			parent = arr
			current = arr[index]
			continue
		}

		// Handle object key notation
		if i == lastIdx {
			switch p := current.(type) {
			case data.Map:
				p[segment] = value
				return root, nil
			default:
				return nil, fmt.Errorf("cannot set key '%s' on non-object value", segment)
			}
		}

		// Initialize or navigate through map
		if current == nil {
			current = data.Map{}
		}
		m, ok := current.(data.Map)
		if !ok {
			return nil, fmt.Errorf("expected object at path segment '%s', got %T", segment, current)
		}

		next := m[segment]
		if next == nil {
			next = data.Map{}
			m[segment] = next
		}
		parent = m
		current = next
	}

	return root, nil
}
