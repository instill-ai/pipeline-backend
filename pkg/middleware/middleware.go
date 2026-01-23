package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

func HandleProfileImage(srv service.Service, repo repository.Repository) runtime.HandlerFunc {

	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		ns, err := srv.GetNamespaceByID(ctx, pathParams["namespaceID"])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pipelineID := pathParams["pipelineID"]
		dbModel, err := repo.GetPipelineByID(ctx, ns.Permalink(), pipelineID, true, true)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if dbModel.ProfileImage.String == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		profileImageBase64 := dbModel.ProfileImage.String

		b, err := base64.StdEncoding.DecodeString(strings.Split(profileImageBase64, ",")[1])
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/png")
		_, err = w.Write(b)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})
}
