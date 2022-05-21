package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/x/zapadapter"

	worker "github.com/instill-ai/vdp/pkg/temporal"
)

var c client.Client

// assign global client for Temporal server
func Init() {
	logger, _ := logger.GetZapLogger()

	// The client is a heavyweight object that should be created once per process.
	var err error

	clientOpts := config.Config.Temporal.ClientOptions
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

func TriggerTemporalWorkflow(pipelineName string, recipe *datamodel.Recipe, uid string, userName string) (map[string]string, error) {
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
	if err := we.Get(context.Background(), &workflowResult); err != nil {
		return nil, err
	}

	return workflowResult, nil
}

// Direct trigger: DS/DD kind are HTTP and no LO defined
// NOTE: Before migrate inference-backend into pipeline-backend, there is one more criteria is only 1 VDO
// func IsDirect(recipe *datamodel.Recipe) bool {

// 	return (strings.ToLower(recipe.Source.Name) == definition.DataSourceKindDirect &&
// 		strings.ToLower(recipe.Destination.Name) == definition.DataDestinationKindDirect &&
// 		len(recipe.Models) == 1 && (recipe.Logics == nil || len(recipe.Logics) == 0))
// }

func recipeToDSLConfig(recipe *datamodel.Recipe, requestId string) worker.Workflow {
	logger, _ := logger.GetZapLogger()

	dslConfigVariables := make(map[string]string)
	var rootSequenceElement []*worker.Statement

	// Extracting data source
	logger.Debug(fmt.Sprintf("The data source configuration is: %+v", recipe.Source))

	// Extracting visual data operator
	logger.Debug(fmt.Sprintf("The visual data operator configuration is: %+v", recipe.ModelInstances))
	for _, modelInstance := range recipe.ModelInstances {
		visualDataOpActivity := worker.ActivityInvocation{
			Name:      "VisualDataOperatorActivity",
			Arguments: []string{"VDOModelId", "VDOVersion", "VDORequestId"},
			Result:    "visualDataOperatorResult",
		}

		// TODO: Revisit here to implement with model-backend
		dslConfigVariables["ModelName"] = modelInstance + "-model-name"
		dslConfigVariables["ModelInstanceName"] = modelInstance + "-model-instance-name"
		dslConfigVariables["VDORequestId"] = requestId

		rootSequenceElement = append(rootSequenceElement, &worker.Statement{Activity: &visualDataOpActivity})
	}

	// Extracting logic operator
	logger.Debug(fmt.Sprintf("The data logic operator configuration is: %+v", recipe.Logics))

	// Extracting data destination
	logger.Debug(fmt.Sprintf("The data destination configuration is: %+v", recipe.Destination))

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
