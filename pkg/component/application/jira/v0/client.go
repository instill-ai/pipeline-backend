package jira

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"

	errorsx "github.com/instill-ai/x/errors"
)

const apiBaseURL = "https://api.atlassian.com"

type client struct {
	*resty.Client
	APIBaseURL string `json:"api-base-url"`
	Domain     string `json:"domain"`
	CloudID    string `json:"cloud-id"`
}

type cloudID struct {
	ID string `json:"cloudId"`
}

type authConfig struct {
	Email   string `json:"email"`
	Token   string `json:"token"`
	BaseURL string `json:"base-url"`
}

func newClient(_ context.Context, setup *structpb.Struct, logger *zap.Logger) (*client, error) {
	var authConfig authConfig
	if err := base.ConvertFromStructpb(setup, &authConfig); err != nil {
		return nil, err
	}

	email := authConfig.Email
	token := authConfig.Token
	baseURL := authConfig.BaseURL
	if token == "" {
		return nil, errorsx.AddMessage(
			fmt.Errorf("token not provided"),
			"token not provided",
		)
	}
	if email == "" {
		return nil, errorsx.AddMessage(
			fmt.Errorf("email not provided"),
			"email not provided",
		)
	}
	cloudID, err := getCloudID(baseURL)
	if err != nil {
		return nil, err
	}

	return &client{
		Client: httpclient.New(
			"Jira-Client",
			baseURL,
			httpclient.WithLogger(logger),
			httpclient.WithEndUserError(new(errBody)),
		).SetHeader("Accept", "application/json").
			SetHeader("Content-Type", "application/json").
			SetBasicAuth(email, token),
		APIBaseURL: apiBaseURL,
		Domain:     baseURL,
		CloudID:    cloudID,
	}, nil
}

func getCloudID(baseURL string) (string, error) {
	client := httpclient.New("Get-Domain-ID", baseURL, httpclient.WithEndUserError(new(errBody)))
	resp := cloudID{}
	req := client.R().SetResult(&resp)
	// See https://developer.atlassian.com/cloud/jira/software/rest/intro/#base-url-differences
	if _, err := req.Get("_edge/tenant_info"); err != nil {
		return "", err
	}
	return resp.ID, nil
}

type errBody struct {
	Body struct {
		Msg []string `json:"errorMessages"`
	} `json:"body"`
}

func (e errBody) Message() string {
	return strings.Join(e.Body.Msg, " ")
}

func turnToStringQueryParams(val any) string {
	var stringVal string
	switch val := val.(type) {
	case string:
		stringVal = val
	case int:
		stringVal = fmt.Sprintf("%d", val)
	case bool:
		stringVal = fmt.Sprintf("%t", val)
	case []string:
		stringVal = strings.Join(val, ",")
	case []int:
		var strVals []string
		for _, v := range val {
			strVals = append(strVals, fmt.Sprintf("%d", v))
		}
		stringVal = strings.Join(strVals, ",")
	default:
		return ""
	}
	return stringVal
}

func addQueryOptions(req *resty.Request, opt interface{}) error {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			if !v.MapIndex(key).IsValid() || !v.MapIndex(key).CanInterface() {
				continue
			}
			val := v.MapIndex(key).Interface()
			stringVal := turnToStringQueryParams(val)
			if stringVal == fmt.Sprintf("%v", reflect.Zero(reflect.TypeOf(val))) {
				continue
			}
			paramName := key.String()
			req.SetQueryParam(paramName, stringVal)
		}
	} else if v.Kind() == reflect.Struct {
		typeOfS := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsValid() || !v.Field(i).CanInterface() {
				continue
			}
			val := v.Field(i).Interface()
			stringVal := turnToStringQueryParams(val)
			if stringVal == fmt.Sprintf("%v", reflect.Zero(reflect.TypeOf(val))) {
				continue
			}
			paramName := typeOfS.Field(i).Tag.Get("api")
			if paramName == "" {
				paramName = typeOfS.Field(i).Name
			}
			req.SetQueryParam(paramName, stringVal)
		}
	}
	return nil
}
