package slack

import (
	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"
)

func newClient(setup *structpb.Struct) *slack.Client {
	return slack.New(getToken(setup))
}

// Need to confirm where the map is
func getToken(setup *structpb.Struct) string {
	return setup.GetFields()["token"].GetStringValue()
}
