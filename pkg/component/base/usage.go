package base

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
)

// UsageHandler allows the component execution wrapper to add checks and
// collect usage metrics around a component execution.
type UsageHandler interface {
	Check(ctx context.Context, inputs []*structpb.Struct) error
	Collect(ctx context.Context, inputs, outputs []*structpb.Struct) error
}

// UsageHandlerCreator returns a function to initialize a UsageHandler.
type UsageHandlerCreator func(IExecution) (UsageHandler, error)

type noopUsageHandler struct{}

func (h *noopUsageHandler) Check(context.Context, []*structpb.Struct) error          { return nil }
func (h *noopUsageHandler) Collect(_ context.Context, _, _ []*structpb.Struct) error { return nil }

// NewNoopUsageHandler is a no-op usage handler initializer.
func NewNoopUsageHandler(IExecution) (UsageHandler, error) {
	return new(noopUsageHandler), nil
}
