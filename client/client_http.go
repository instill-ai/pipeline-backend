package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	url := "http://localhost:8081/v1beta/users/admin/pipelines/test1/trigger"
	method := "POST"

	payload := []byte(`{"inputs":[{"input":"tett"}]}`)

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

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println(string(body))
}
