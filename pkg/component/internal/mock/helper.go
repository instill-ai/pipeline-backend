package mock

import (
	"fmt"
	"reflect"

	"github.com/gojuno/minimock/v3"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// GenerateMockJob creates a base.Job with injected mocks, which are returned
// along with the job.
func GenerateMockJob(c minimock.Tester) (ir *InputReaderMock, ow *OutputWriterMock, eh *ErrorHandlerMock, job *base.Job) {
	ir = NewInputReaderMock(c)
	ow = NewOutputWriterMock(c)
	eh = NewErrorHandlerMock(c)
	job = &base.Job{
		Input:  ir,
		Output: ow,
		Error:  eh,
	}

	return
}

// Equal panics if got is not equal to want.
func Equal[T comparable](got, want T) {
	if got != want {
		panic(fmt.Sprintf("Expected %v, but got %v", want, got))
	}
}

// NotNil panics if value is nil.
func NotNil(value interface{}) {
	if value == nil {
		panic("Expected value to be not nil")
	}
}

// Nil panics if value is not nil.
func Nil(value interface{}) {
	if value != nil {
		panic(fmt.Sprintf("Expected value to be nil, but got %v", value))
	}
}

// Contains panics if slice does not contain the expected element.
func Contains[T comparable](slice []T, element T) {
	found := false
	for _, item := range slice {
		if item == element {
			found = true
			break
		}
	}
	if !found {
		panic(fmt.Sprintf("Expected slice to contain element %v, but it was not found in %v", element, slice))
	}
}

// DeepEquals uses reflect.DeepEqual to compare complex types.
// Panics if got and want are not deeply equal.
func DeepEquals(got, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		panic(fmt.Sprintf("Expected %#v, but got %#v", want, got))
	}
}
