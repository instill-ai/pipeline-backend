package handler

import (
	"context"
	"fmt"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"net/http"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	// pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey) == "user" {
		if resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	} else {
		if !allowVisitor {
			return service.ErrUnauthenticated
		}
		if resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session UUID from the request URL
	sessionUUID := r.URL.Path[len("/sse/"):]

	// Get the data channel for the session UUID
	dataChanValue, ok := middleware.DataChanMap.Load(sessionUUID)
	if !ok {
		http.Error(w, "Invalid session UUID", http.StatusBadRequest)
		return
	}
	dataChan, ok := dataChanValue.(chan []byte)
	if !ok {
		http.Error(w, "Invalid data channel", http.StatusInternalServerError)
		return
	}

	// Set the response headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var lastTimestamp int64 = 0
	var eventIDCounter int64 = 0

	// Send the data chunks as SSE events
	for data := range dataChan {
		timestamp := time.Now().UnixNano()
		if timestamp == lastTimestamp {
			eventIDCounter++
		} else {
			eventIDCounter = 0
		}
		lastTimestamp = timestamp

		fmt.Fprintf(w, "event: output\n")
		fmt.Fprintf(w, "id: %d:%d\n", timestamp, eventIDCounter)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Send the "done" event
	currentTimestamp := time.Now().UnixNano()
	if currentTimestamp == lastTimestamp {
		eventIDCounter++
	} else {
		eventIDCounter = 0
	}

	fmt.Fprintf(w, "event: done\n")
	fmt.Fprintf(w, "id: %d:%d\n", currentTimestamp, eventIDCounter)
	fmt.Fprintf(w, "data: {}\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Remove the data channel from the map when the SSE connection is closed
	middleware.DataChanMap.Delete(sessionUUID)
}
