//go:build integration
// +build integration

package ai

import (
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/ai/openai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

var (
	// TODO read keys from env variables and check for valid response.
	openAIKey = "invalid-key"
	openAIOrg = "no-such-org"

	emptyOptions = ComponentOptions{}
)

func TestOpenAITextGeneration(t *testing.T) {
	c := qt.New(t)

	config, err := structpb.NewStruct(map[string]any{
		"api-key":      openAIKey,
		"organization": openAIOrg,
	})
	c.Assert(err, qt.IsNil)

	in, err := base.ConvertToStructpb(openai.TextCompletionInput{
		Prompt: "how are you doing?",
		Model:  "gpt-3.5-turbo",
	})
	c.Assert(err, qt.IsNil)

	logger := zap.NewNop()
	conn := Init(logger, emptyOptions)

	def, err := conn.GetComponentDefinitionByID("openai", nil)
	c.Assert(err, qt.IsNil)

	uid, err := uuid.FromString(def.GetUid())
	c.Assert(err, qt.IsNil)

	exec, err := conn.CreateExecution(uid, "TASK_TEXT_GENERATION", config, logger)
	c.Assert(err, qt.IsNil)

	op, err := exec.Execution.Execute([]*structpb.Struct{in})

	// This assumes invalid API credentials that trigger a 401 API error. We
	// should check for valid behaviour in integration tests, although
	// validating the format of error messages can also be useful to detect
	// breaking changes in the API.
	c.Check(err, qt.ErrorMatches, "unsuccessful HTTP response")
	c.Check(errmsg.Message(err), qt.Matches, ".*Incorrect API key provided.*")

	c.Logf("op: %v", op)
	c.Logf("err: %v", err)
	c.Log("end-user err:", errmsg.MessageOrErr(err))
}
