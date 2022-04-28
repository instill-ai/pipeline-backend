package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/proto"
)

func httpResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	// set http status code
	if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
		code, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		// delete the headers to not expose any grpc-metadata in http response
		delete(md.HeaderMD, "x-http-code")
		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
		delete(w.Header(), "Grpc-Metadata-Content-Type")
		delete(w.Header(), "Grpc-Metadata-Trailer")
		w.WriteHeader(code)
	}

	return nil
}

func customMatcher(key string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(key), "jwt-") {
		return key, true
	}

	switch key {
	case "owner_id":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}
