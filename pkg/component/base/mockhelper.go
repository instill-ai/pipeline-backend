package base

import (
	"github.com/gojuno/minimock/v3"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func GenerateMockJob(c minimock.Tester) (ir *mock.InputReaderMock, ow *mock.OutputWriterMock, eh *mock.ErrorHandlerMock, job *Job) {
	ir = mock.NewInputReaderMock(c)
	ow = mock.NewOutputWriterMock(c)
	eh = mock.NewErrorHandlerMock(c)
	job = &Job{
		Input:  ir,
		Output: ow,
		Error:  eh,
	}
	return
}
