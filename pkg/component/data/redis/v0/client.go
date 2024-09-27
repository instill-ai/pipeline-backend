package redis

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	goredis "github.com/redis/go-redis/v9"
)

func getHost(setup *structpb.Struct) string {
	return setup.GetFields()["host"].GetStringValue()
}
func getPort(setup *structpb.Struct) int {
	return int(setup.GetFields()["port"].GetNumberValue())
}
func getPassword(setup *structpb.Struct) string {
	val, ok := setup.GetFields()["password"]
	if !ok {
		return ""
	}
	return val.GetStringValue()
}
func getUsername(setup *structpb.Struct) string {
	val, ok := setup.GetFields()["username"]
	if !ok {
		return ""
	}
	return val.GetStringValue()
}

func getSSL(setup *structpb.Struct) bool {
	val, ok := setup.GetFields()["ssl"]
	if !ok {
		return false
	}
	return val.GetBoolValue()
}

// NewClient creates a new redis client
func NewClient(setup *structpb.Struct) (*goredis.Client, error) {
	op := &goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", getHost(setup), getPort(setup)),
		Password: getPassword(setup),
		DB:       0,
	}
	if getUsername(setup) != "" {
		op.Username = getUsername(setup)
	}

	if getSSL(setup) {
		op.TLSConfig = &tls.Config{}
	}

	// TODO - add SSH support

	return goredis.NewClient(op), nil
}
