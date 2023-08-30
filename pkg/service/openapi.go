package service

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

const openApiSchemaTemplate = `{
    "openapi": "3.0.0",
    "info": {
        "version": "dev",
        "title": "Pipeline Trigger"
    },
    "paths": {
        "/trigger": {
            "post": {
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "inputs": {
                                        "type": "array",
                                        "items": {}
                                    }
                                }
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "outputs": {
                                            "type": "array",
                                            "items": {}
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
		"/triggerAsync": {
            "post": {
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "inputs": {
                                        "type": "array",
                                        "items": {}
                                    }
                                }
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "",
                        "content": {
                            "application/json": {
                                "schema": {
									"type": "object",
									"properties": {
										"operation": {
											"title": "Trigger async pipeline operation message",
											"readOnly": true,
											"type": "object"
										}
									}
								}
                            }
                        }
                    }
                }
            }
        }
    }
}`

func (s *service) GenerateOpenApiSpec(startComp *pipelinePB.Component, endComp *pipelinePB.Component) (*structpb.Struct, error) {
	success := true
	template := &structpb.Struct{}
	err := protojson.Unmarshal([]byte(openApiSchemaTemplate), template)
	if err != nil {
		return nil, err
	}

	var walk *structpb.Value

	openApiInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openApiInput.Fields["type"] = structpb.NewStringValue("object")
	openApiInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range startComp.Configuration.Fields["body"].GetStructValue().Fields {
		var m *structpb.Value
		attrType := ""
		arrType := ""
		switch t := v.GetStructValue().Fields["type"].GetStringValue(); t {
		case "integer", "number", "boolean":
			attrType = t
		case "text", "image", "audio", "video":
			attrType = "string"
		default:
			attrType = "array"
			switch t2 := v.GetStructValue().Fields["type"].GetStringValue(); t {
			case "integer_array", "number_array", "boolean_array":
				arrType = strings.Split(t2, "_")[0]
			case "text_array", "image_array", "audio_array", "video_array":
				arrType = "string"
			}

		}
		if attrType != "array" {
			m, err = structpb.NewValue(map[string]interface{}{
				"title":       v.GetStructValue().Fields["title"].GetStringValue(),
				"description": v.GetStructValue().Fields["description"].GetStringValue(),
				"type":        attrType,
			})
			if err != nil {
				success = false
			}
		} else {
			m, err = structpb.NewValue(map[string]interface{}{
				"title":       v.GetStructValue().Fields["title"].GetStringValue(),
				"description": v.GetStructValue().Fields["description"].GetStringValue(),
				"type":        attrType,
				"items": map[string]interface{}{
					"type": arrType,
				},
			})
			if err != nil {
				success = false
			}
		}

		openApiInput.Fields["properties"].GetStructValue().Fields[k] = m

	}

	walk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		walk = walk.GetStructValue().Fields[key]
	}
	*walk = *structpb.NewStructValue(openApiInput)
	walk = template.GetFields()["paths"]
	for _, key := range []string{"/triggerAsync", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		walk = walk.GetStructValue().Fields[key]
	}
	*walk = *structpb.NewStructValue(openApiInput)

	// output

	openApiOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openApiOutput.Fields["type"] = structpb.NewStringValue("object")
	openApiOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range endComp.Configuration.Fields["body"].GetStructValue().Fields {
		var m *structpb.Value

		m, err = structpb.NewValue(map[string]interface{}{
			"title":       v.GetStructValue().Fields["title"].GetStringValue(),
			"description": v.GetStructValue().Fields["description"].GetStringValue(),
			// "type":        attrType,
		})

		if err != nil {
			success = false
		}

		openApiOutput.Fields["properties"].GetStructValue().Fields[k] = m

	}

	walk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
		walk = walk.GetStructValue().Fields[key]
	}
	*walk = *structpb.NewStructValue(openApiOutput)

	if success {
		return template, nil
	}
	return nil, fmt.Errorf("generate OpenAPI spec error")

}
