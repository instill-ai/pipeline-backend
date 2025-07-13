package base

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	errorsx "github.com/instill-ai/x/errors"
)

// IExecution allows components to be executed.
type IExecution interface {
	GetTask() string
	GetLogger() *zap.Logger
	GetSystemVariables() map[string]any
	GetComponent() IComponent
	GetComponentID() string
	UsesInstillCredentials() bool

	Execute(context.Context, []*Job) error
}

// Job is the job for the component.
type Job struct {
	Input  InputReader
	Output OutputWriter
	Error  ErrorHandler
}

// InputReader is an interface for reading input data from a job.
type InputReader interface {
	// ReadData reads the input data from the job into the provided struct.
	ReadData(ctx context.Context, input any) (err error)

	// Deprecated: Read() is deprecated and will be removed in a future version.
	// Use ReadData() instead. structpb is not suitable for handling binary data
	// and will be phased out gradually.
	Read(ctx context.Context) (input *structpb.Struct, err error)
}

// OutputWriter is an interface for writing output data to a job.
type OutputWriter interface {
	// WriteData writes the output data to the job from the provided struct.
	WriteData(ctx context.Context, output any) (err error)

	// Deprecated: Write() is deprecated and will be removed in a future
	// version. Use WriteData() instead. structpb is not suitable for handling
	// binary data and will be phased out gradually.
	Write(ctx context.Context, output *structpb.Struct) (err error)
}

// ErrorHandler is an interface for handling errors from a job.
type ErrorHandler interface {
	Error(ctx context.Context, err error)
}

// ComponentExecution implements the common methods for component execution.
type ComponentExecution struct {
	Component IComponent

	// Component ID is the ID of the component *as defined in the recipe*. This
	// identifies an instance of a component, which holds a given configuration
	// (task, setup, input parameters, etc.).
	//
	// NOTE: this is a property of the component not of the execution. However,
	// right now components are being created on startup and only executions
	// are created every time a pipeline is triggered. Therefore, at the moment
	// there's no intermediate entity reflecting "a component within a
	// pipeline". Since we need to access the component ID for e.g. logging /
	// metric collection purposes, for now this information will live in the
	// execution, but note that several executions might have the same
	// component ID.
	ComponentID     string
	SystemVariables map[string]any
	Setup           *structpb.Struct
	Task            string
}

// GetComponent returns the component interface that is triggering the execution.
func (e *ComponentExecution) GetComponent() IComponent { return e.Component }

// GetComponentID returns the ID of the component that's being executed.
func (e *ComponentExecution) GetComponentID() string { return e.ComponentID }

// GetTask returns the task that the component is executing.
func (e *ComponentExecution) GetTask() string { return e.Task }

// GetSetup returns the setup of the component.
func (e *ComponentExecution) GetSetup() *structpb.Struct { return e.Setup }

// GetSystemVariables returns the system variables of the component.
func (e *ComponentExecution) GetSystemVariables() map[string]any { return e.SystemVariables }

// GetLogger returns the logger of the component.
func (e *ComponentExecution) GetLogger() *zap.Logger { return e.Component.GetLogger() }

// UsesInstillCredentials indicates whether the component setup includes the use
// of global secrets (as opposed to a bring-your-own-key configuration) to
// connect to external services. Components should override this method when
// they have the ability to read global secrets and be executed without
// explicit credentials.
func (e *ComponentExecution) UsesInstillCredentials() bool { return false }

// SecretKeyword is a keyword to reference a secret in a component
// configuration. When a component detects this value in a configuration
// parameter, it will used the pre-configured value, injected at
// initialization.
const SecretKeyword = "__INSTILL_SECRET"

// NewUnresolvedCredential returns an end-user error signaling that the
// component setup contains credentials that reference a global secret that
// wasn't injected into the component.
func NewUnresolvedCredential(key string) error {
	return errorsx.AddMessage(
		fmt.Errorf("unresolved global credential"),
		fmt.Sprintf("The configuration field %s references a global secret "+
			"but it doesn't support Instill Credentials.", key),
	)
}
