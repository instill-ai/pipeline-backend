package anthropic

import (
	"google.golang.org/protobuf/types/known/structpb"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	anthropicsdkoption "github.com/anthropics/anthropic-sdk-go/option"
)

func newClient(apiKey string) *anthropicsdk.Client {
	client := anthropicsdk.NewClient(
		anthropicsdkoption.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	return client

}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}
