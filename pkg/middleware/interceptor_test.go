package middleware

import (
	"fmt"
	"testing"

	"github.com/jackc/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
)

func TestAsGRPCError(t *testing.T) {
	c := qt.New(t)

	c.Run("nil", func(c *qt.C) {
		c.Assert(AsGRPCError(nil), qt.IsNil)
	})

	testcases := []struct {
		name        string
		in          error
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name: "unknown",
			in: &pgconn.PgError{
				Severity: "FATAL",
				Code:     "08006",
				Message:  "connection_failure",
				Detail:   "connection_failure",
			},
			wantCode:    codes.Unknown,
			wantMessage: ".*FATAL.*connection_failure.*",
		},
		{
			name:        "resource exists",
			in:          errdomain.ErrAlreadyExists,
			wantCode:    codes.AlreadyExists,
			wantMessage: "Resource already exists.",
		},
		{
			name:        "already a gRPC status",
			in:          status.Error(codes.FailedPrecondition, "pipeline recipe error"),
			wantCode:    codes.FailedPrecondition,
			wantMessage: "pipeline recipe error",
		},
		{
			name: "gRPC status with end-user message",
			in: errmsg.AddMessage(
				status.Error(codes.FailedPrecondition, "pipeline recipe error"),
				"Invalid recipe in pipeline",
			),
			wantCode:    codes.FailedPrecondition,
			wantMessage: "Invalid recipe in pipeline",
		},
		{
			name:        "not found",
			in:          fmt.Errorf("finding item: %w", errdomain.ErrNotFound),
			wantCode:    codes.NotFound,
			wantMessage: "finding item: not found",
		},
		{
			name:        "unauthorized",
			in:          fmt.Errorf("checking requester permission: %w", errdomain.ErrUnauthorized),
			wantCode:    codes.PermissionDenied,
			wantMessage: "checking requester permission: unauthorized",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			got := AsGRPCError(tc.in)
			c.Assert(got, qt.IsNotNil)

			st, ok := status.FromError(got)
			c.Assert(ok, qt.IsTrue)
			c.Assert(st.Code(), qt.Equals, tc.wantCode)
			c.Assert(st.Message(), qt.Matches, tc.wantMessage)
		})
	}
}
