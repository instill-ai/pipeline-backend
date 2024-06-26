package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Response struct {
	SessionUUID      string `json:"SessionUUID"`
	SourceInstanceID string `json:"SourceInstanceID"`
}

func main() {
	useSSE := true
	url := "http://localhost:8081/v1beta/users/admin/pipelines/test1/trigger:stream"
	method := "POST"

	payload := []byte(`{"inputs":[{"input":"test"}]}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Language", "en-GB")
	req.Header.Add("Access-Control-Allow-Headers", "instill-return-traces, instill-share-code")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/grpc")
	req.Header.Add("DNT", "1")
	req.Header.Add("Origin", "http://localhost:3000")
	req.Header.Add("Referer", "http://localhost:3000/")
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-site")
	req.Header.Add("User-Agent", "grpc-go/1.64.0")
	req.Header.Add("instill-return-traces", "true")
	req.Header.Add("instill-user-uid", "8efcaa75-2522-4b06-9a5c-63df9fb7351c")
	req.Header.Add("grpcgateway-user-agent", "KrakenD Version 2.6.2")
	req.Header.Add("content-type", "application/grpc")
	req.Header.Add("jwt-sub", "8efcaa75-2522-4b06-9a5c-63df9fb7351c")
	req.Header.Add("x-forwarded-host", "localhost:8080")
	req.Header.Add("x-forwarded-for", "172.18.0.1, 172.18.0.13")
	req.Header.Add("x-b3-spanid", "0d47d86f57e7cfd0")
	req.Header.Add("x-b3-traceid", "de165dfd6275ea5f18840f7a78889838")
	req.Header.Add("grpcgateway-content-type", "application/json")
	req.Header.Add("x-b3-sampled", "1")
	req.Header.Add("instill-auth-type", "user")
	req.Header.Add("grpc-accept-encoding", "gzip")
	if useSSE {
		req.Header.Add("X-Use-SSE", "true")

	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer res.Body.Close()

	buf := make([]byte, 1024)
	var responseBody []byte
	for {
		n, err := res.Body.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("Error reading response: %v", err)
		}
		if n == 0 {
			break
		}
		fmt.Print(string(buf[:n]) + "\n\n")
		responseBody = append(responseBody, buf[:n]...)
		time.Sleep(time.Millisecond)
	}
	var response Response
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		log.Fatalf("Error unmarshaling response: %v", err)
	}

	fmt.Printf("SessionUUID: %s\n", response.SessionUUID)
	fmt.Printf("SourceInstanceID: %s\n", response.SourceInstanceID)

	sseURL := fmt.Sprintf("http://localhost:8081/sse/%s", response.SessionUUID)
	sseReq, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		log.Fatalf("Error creating SSE request: %v", err)
	}

	sseReq.Header.Add("Accept", "text/event-stream")
	sseReq.Header.Add("instill-user-uid", "8efcaa75-2522-4b06-9a5c-63df9fb7351c")
	req.Header.Add("Access-Control-Allow-Headers", "instill-return-traces, instill-share-code")

	sseRes, err := client.Do(sseReq)
	if err != nil {
		log.Fatalf("Error sending SSE request: %v", err)
	}
	defer sseRes.Body.Close()

	buf = make([]byte, 1024)
	for {
		n, err := sseRes.Body.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("Error reading SSE response: %v", err)
		}
		if n == 0 {
			break
		}
		fmt.Print(string(buf[:n]) + "\n\n")
		time.Sleep(time.Millisecond)
	}
}

//curl -X POST "http://localhost:8081/v1beta/users/admin/pipelines/test1/trigger:stream" -H "Accept: application/json, text/plain, */*" -H "Accept-Language: en-GB" -H "Access-Control-Allow-Headers: instill-return-traces, instill-share-code" -H "Connection: keep-alive" -H "instill-return-traces: true" -H "instill-user-uid: 8efcaa75-2522-4b06-9a5c-63df9fb7351c" -H "jwt-sub: 8efcaa75-2522-4b06-9a5c-63df9fb7351c" -H "x-b3-sampled: 1" -H "instill-auth-type: user" -H "X-Use-SSE: true" --no-buffer --header "Transfer-Encoding: chunked" --header "Content-Type: application/json" -d '{"inputs":[{"input":"test"}]}'
//
//curl -X POST "http://localhost:8081/v1beta/users/admin/pipelines/test1/trigger:stream" \
//-H "Accept: application/json, text/plain, */*" \
//-H "Accept: text/event-stream" \
//http://localhost:8081/sse/336878eec294bb0fac404f096d878d2cda35f325b585f973e14d1c16243962be
