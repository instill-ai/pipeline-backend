package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/instill-ai/pipeline-backend/internal/constant"
	"github.com/instill-ai/pipeline-backend/internal/db"
	"github.com/instill-ai/pipeline-backend/internal/external"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

func errorResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {

		owner := req.Header.Get("owner")
		id := pathParams["id"]

		if owner == "" {
			errorResponse(w, 400, "Bad Request", "Required parameter Jwt-Sub not found in the header")
			return
		}

		if id == "" {
			errorResponse(w, 400, "Bad Request", "Required parameter pipeline id not found in the path")
			return
		}

		service := service.NewService(
			repository.NewRepository(db.GetConnection()),
			external.InitConnectorServiceClient(),
			external.InitModelServiceClient(),
		)

		dbPipeline, err := service.GetPipelineByID(id, owner, false)
		if err != nil {
			errorResponse(w, 400, "Bad Request", "Pipeline not found")
			return
		}

		if err := req.ParseMultipartForm(4 << 20); err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}

		fileBytes, fileLengths, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading files from request")
			return
		}

		var obj interface{}
		if obj, err = service.TriggerPipelineBinaryFileUpload(*bytes.NewBuffer(fileBytes), fileLengths, dbPipeline); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		ret, _ := json.Marshal(obj)
		_, _ = w.Write(ret)
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(405)
	}
}

func parseImageFormDataInputsToBytes(req *http.Request) (fileBytes []byte, fileLengths []uint64, err error) {
	inputs := req.MultipartForm.File["file"]
	var file multipart.File
	for _, content := range inputs {
		file, err = content.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			return nil, nil, fmt.Errorf("Unable to open file for image")
		}

		buff := new(bytes.Buffer)
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to read content body from image")
		}

		if numBytes > int64(constant.MaxImageSizeBytes) {
			return nil, nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB",
				float32(constant.MaxImageSizeBytes)/float32(constant.MB),
				float32(numBytes)/float32(constant.MB),
			)
		}

		fileBytes = append(fileBytes, buff.Bytes()...)
		fileLengths = append(fileLengths, uint64(buff.Len()))
	}

	return fileBytes, fileLengths, nil
}
