package temporal

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bochengyang/zapadapter"
	"github.com/instill-ai/pipeline-backend/configs"
	"github.com/instill-ai/pipeline-backend/internal/definition"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	worker "github.com/instill-ai/visual-data-pipeline/pkg/temporal"
	"go.temporal.io/sdk/client"
)

var c client.Client

// assign global client for Temporal server
func Init() {
	logger, _ := logger.GetZapLogger()

	// The client is a heavyweight object that should be created once per process.
	var err error

	clientOpts := configs.Config.Temporal.ClientOptions
	clientOpts.Logger = zapadapter.NewZapAdapter(logger)

	c, err = client.NewClient(clientOpts)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to create client: %s", err))
	}
}

func Close() {
	if c != nil {
		c.Close()
	}
}

func TriggerTemporalWorkflow(pipelineName string, recipe *model.Recipe, uid string, userName string) (map[string]string, error) {
	logger, _ := logger.GetZapLogger()

	dslWorkflow := recipeToDSLConfig(recipe, uid)

	workflowId := fmt.Sprintf("%s_%s", userName, pipelineName)

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: worker.TaskQueueName,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, worker.PipelineWorkflow, dslWorkflow)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to execute workflow: %s", err))
	}

	logger.Info(fmt.Sprintf("Started workflow WorkflowID %s RunID %s", we.GetID(), we.GetRunID()))

	logger.Info("Awaiting workflow finished")

	// Use the WorkflowExecution to get the result
	// Get is blocking call and will wait for the Workflow to complete
	var workflowResult map[string]string
	we.Get(context.Background(), &workflowResult)

	return workflowResult, nil
}

// Direct trigger: DS/DD kind are HTTP and no LO defined
// NOTE: Before migrate inference-backend into pipeline-backend, there is one more criteria is only 1 VDO
func IsDirect(recipe *model.Recipe) bool {

	return (strings.ToLower(recipe.DataSource.Type) == definition.DataSourceKindHTTP &&
		strings.ToLower(recipe.DataDestination.Type) == definition.DataDestinationKindHTTP &&
		len(recipe.VisualDataOperator) == 1 &&
		(recipe.LogicOperator == nil || len(recipe.LogicOperator) == 0))
}

func recipeToDSLConfig(recipe *model.Recipe, requestId string) worker.Workflow {
	logger, _ := logger.GetZapLogger()

	dslConfigVariables := make(map[string]string)
	var rootSequenceElement []*worker.Statement

	// Extracting data source
	logger.Debug(fmt.Sprintf("The data source configuration is: %+v", recipe.DataSource))

	// Extracting visual data operator
	logger.Debug(fmt.Sprintf("The visual data operator configuration is: %+v", recipe.VisualDataOperator))
	for _, vdo := range recipe.VisualDataOperator {
		visualDataOpActivity := worker.ActivityInvocation{
			Name:      "VisualDataOperatorActivity",
			Arguments: []string{"VDOModelId", "VDOVersion", "VDORequestId"},
			Result:    "visualDataOperatorResult",
		}

		dslConfigVariables["VDOModelId"] = vdo.ModelId
		dslConfigVariables["VDOVersion"] = strconv.FormatInt(int64(vdo.Version), 10)
		dslConfigVariables["VDORequestId"] = requestId

		rootSequenceElement = append(rootSequenceElement, &worker.Statement{Activity: &visualDataOpActivity})
	}

	// Extracting logic operator
	logger.Debug(fmt.Sprintf("The data logic operator configuration is: %+v", recipe.LogicOperator))

	// Extracting data destination
	logger.Debug(fmt.Sprintf("The data destination configuration is: %+v", recipe.DataDestination))

	return worker.Workflow{
		Variables: dslConfigVariables,
		Root: worker.Statement{
			Activity: nil,
			Parallel: nil,
			Sequence: &worker.Sequence{
				Elements: rootSequenceElement,
			},
		},
	}
}
