package main

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"io"
	"log"
	"time"
)

// Define a sample TriggerData struct
type TriggerData struct {
	Key   string
	Value string
}

func main() {

	// Create a new context
	ctx := context.Background()

	// Create a new metadata map
	md := metadata.New(map[string]string{
		"instill-return-traces":     "true",
		"instill-user-uid":          "8efcaa75-2522-4b06-9a5c-63df9fb7351c",
		"user-agent":                "grpc-go/1.64.0",
		"grpcgateway-user-agent":    "KrakenD Version 2.6.2",
		"grpcgateway-authorization": "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6Imluc3RpbGwiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjE3MTg2Njg0NzgsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6ODA4MCIsImp0aSI6IjEzOTM1ZDdlLWY1ZmQtNDk1NS04YTVhLTM4MWY5OWNkMmU2MiIsInN1YiI6IjhlZmNhYTc1LTI1MjItNGIwNi05YTVjLTYzZGY5ZmI3MzUxYyJ9.Z2SMgCaozjw0xLYcva3LiWgIcApintpKnGdze4WIkRcimtwHoMLx_qhzIuxIdYdTONUm1VDE557P2uC4DOICikQDXyRlDfPY_4DOnGKBXAAdMfspw1fchMzkQzUZKbWGnJt7pV6PgwLsTt56iYZf2NPVCJftpblcqEbQhhE1bPOgexEqfVC6Ti91wudhhHp0tU5oE6-GDSzfdhb5Bk0ga7-6SzCgmK5ibYj8MceHr-W-sG63eNDh999ic7FhdCde-TEALwHHPdgFeIYp4B4gCVgm8t2wACQRhJmqeDO4oKfDAmpQEtcxocZkI8KkB5YzkCHyb6y2OMWziQEsUxPpNA",
		"content-type":              "application/grpc",
		"jwt-sub":                   "8efcaa75-2522-4b06-9a5c-63df9fb7351c",
		"x-forwarded-host":          "localhost:8080",
		"x-forwarded-for":           "172.18.0.1, 172.18.0.13",
		"x-b3-spanid":               "0d47d86f57e7cfd0",
		"authorization":             "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6Imluc3RpbGwiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjE3MTg2Njg0NzgsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6ODA4MCIsImp0aSI6IjEzOTM1ZDdlLWY1ZmQtNDk1NS04YTVhLTM4MWY5OWNkMmU2MiIsInN1YiI6IjhlZmNhYTc1LTI1MjItNGIwNi05YTVjLTYzZGY5ZmI3MzUxYyJ9.Z2SMgCaozjw0xLYcva3LiWgIcApintpKnGdze4WIkRcimtwHoMLx_qhzIuxIdYdTONUm1VDE557P2uC4DOICikQDXyRlDfPY_4DOnGKBXAAdMfspw1fchMzkQzUZKbWGnJt7pV6PgwLsTt56iYZf2NPVCJftpblcqEbQhhE1bPOgexEqfVC6Ti91wudhhHp0tU5oE6-GDSzfdhb5Bk0ga7-6SzCgmK5ibYj8MceHr-W-sG63eNDh999ic7FhdCde-TEALwHHPdgFeIYp4B4gCVgm8t2wACQRhJmqeDO4oKfDAmpQEtcxocZkI8KkB5YzkCHyb6y2OMWziQEsUxPpNA",
		"x-b3-traceid":              "de165dfd6275ea5f18840f7a78889838",
		"grpcgateway-content-type":  "application/json",
		"x-b3-sampled":              "1",
		":authority":                "localhost:8081",
		"instill-auth-type":         "user",
		"grpc-accept-encoding":      "gzip",
	})

	// Attach the metadata to the context
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPipelinePublicServiceClient(conn)

	jsonData := `{"inputs":[{"input":"test"}]}`

	var rawInputs struct {
		Inputs []map[string]interface{} `json:"inputs"`
	}
	if err := json.Unmarshal([]byte(jsonData), &rawInputs); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	var pbInputs []*structpb.Struct
	for _, input := range rawInputs.Inputs {
		pbStruct, err := structpb.NewStruct(input)
		if err != nil {
			fmt.Println("Error converting to protobuf struct:", err)
			return
		}
		pbInputs = append(pbInputs, pbStruct)
	}

	request := pb.TriggerUserPipelineWithStreamRequest{
		Name:   "users/admin/pipelines/test1",
		Inputs: pbInputs,
	}

	stream, err := c.TriggerUserPipelineWithStream(ctx, &request)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// All responses have been received
			log.Printf("All responses have been received")
			break
		}
		if err != nil {
			log.Fatalf("Error when receiving data: %v", err)
		}
		fmt.Printf("Response received: %v"+"\n\n", response)
		time.Sleep(time.Millisecond * 10)
	}
}
