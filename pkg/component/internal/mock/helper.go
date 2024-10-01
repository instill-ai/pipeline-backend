package mock

import (
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
