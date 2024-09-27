package document

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"code.sajari.com/docconv"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestOperator(t *testing.T) {
	c := qt.New(t)

	fileContent, _ := os.ReadFile("testdata/test.txt")
	base64DataURI := fmt.Sprintf("data:%s;base64,%s", docconv.MimeTypeByExtension("testdata/test.txt"), base64.StdEncoding.EncodeToString(fileContent))

	testcases := []struct {
		name  string
		task  string
		input structpb.Struct
	}{
		{
			name: "convert to text",
			task: "TASK_CONVERT_TO_TEXT",
			input: structpb.Struct{
				Fields: map[string]*structpb.Value{
					"document": {Kind: &structpb.Value_StringValue{StringValue: base64DataURI}},
				},
			},
		},
	}
	bc := base.Component{}
	ctx := context.Background()
	for i := range testcases {
		tc := &testcases[i]
		c.Run(tc.name, func(c *qt.C) {
			component := Init(bc)
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Return(&tc.input, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				return nil
			})
			eh.ErrorMock.Optional()

			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}
