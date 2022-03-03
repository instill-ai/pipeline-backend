package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

func httpReq(method, url string, header map[string][]string, body io.Reader) (int, []byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, body)
	if header != nil {
		req.Header = header
	}
	resp, err := client.Do(req)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.Body == nil {
		return resp.StatusCode, nil, nil
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func Get(endpoint string, header map[string][]string) (int, []byte, error) {
	return httpReq(http.MethodGet, endpoint, header, nil)
}

func MultiPart(endpoint string, header map[string][]string, extParams map[string]string, fileFieldName, fileName string, fileContent []byte) (int, []byte, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileFieldName, fileName)
	if err != nil {
		return 500, nil, err
	}
	if _, err := part.Write(fileContent); err != nil {
		return 500, nil, err
	}

	for key, val := range extParams {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return 500, nil, err
	}

	if header != nil {
		header["Content-Type"] = []string{writer.FormDataContentType()}
	} else {
		header = make(http.Header)
		header["Content-Type"] = []string{writer.FormDataContentType()}
	}

	return httpReq(http.MethodPost, endpoint, header, body)
}
