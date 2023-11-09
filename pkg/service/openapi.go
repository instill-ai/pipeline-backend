package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/utils"
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
						"type": "object",
						"properties": {
						  "name": {
							"type": "string"
						  },
						  "metadata": {
							"type": "object",
							"properties": {
							  "@type": {
								"type": "string"
							  }
							},
							"additionalProperties": {}
						  },
						  "done": {
							"type": "boolean"
						  },
						  "error": {
							"type": "object",
							"properties": {
							  "code": {
								"type": "integer",
								"format": "int32"
							  },
							  "message": {
								"type": "string"
							  },
							  "details": {
								"type": "array",
								"items": {
								  "type": "object",
								  "properties": {
									"@type": {
									  "type": "string"
									}
								  },
								  "additionalProperties": {}
								}
							  }
							}
						  },
						  "response": {
							"type": "object",
							"properties": {
							  "@type": {
								"type": "string"
							  }
							},
							"additionalProperties": {}
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
	  }
	}
  }`

// TODO: refactor these messy code
func (s *service) GenerateOpenApiSpec(startCompOrigin *pipelinePB.Component, endCompOrigin *pipelinePB.Component, compsOrigin []*pipelinePB.Component) (*structpb.Struct, error) {
	success := true
	template := &structpb.Struct{}
	err := protojson.Unmarshal([]byte(openApiSchemaTemplate), template)
	if err != nil {
		return nil, err
	}

	var templateWalk *structpb.Value

	openApiInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openApiInput.Fields["type"] = structpb.NewStringValue("object")
	openApiInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	startComp := proto.Clone(startCompOrigin).(*pipelinePB.Component)
	for k, v := range startComp.Configuration.Fields["metadata"].GetStructValue().Fields {
		openApiInput.Fields["properties"].GetStructValue().Fields[k] = v
	}

	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openApiInput)
	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/triggerAsync", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openApiInput)

	// output

	openApiOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openApiOutput.Fields["type"] = structpb.NewStringValue("object")
	openApiOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	endComp := proto.Clone(endCompOrigin).(*pipelinePB.Component)
	inputFields := endComp.Configuration.Fields["input"].GetStructValue().Fields
	if endComp.Configuration.Fields["metadata"] == nil {
		return nil, fmt.Errorf("metadata of op-end can not be empty")
	}
	for k, v := range endComp.Configuration.Fields["metadata"].GetStructValue().Fields {
		var m *structpb.Value

		var err error

		switch inputFields[k].AsInterface().(type) {
		case string:
			str := inputFields[k].GetStringValue()
			if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") && !strings.HasPrefix(str, "{{") && !strings.HasSuffix(str, "}}") {
				// TODO
				str = str[1:]
				str = str[:len(str)-1]
				str = strings.ReplaceAll(str, " ", "")
				isArrayReference := false
				if str[0] == '[' && str[len(str)-1] == ']' {
					subStrs := strings.Split(str[1:len(str)-1], ",")
					if len(subStrs) == 0 {
						return nil, fmt.Errorf("empty array")
					}
					str = subStrs[0]
					isArrayReference = true
				}

				var b interface{}
				unmarshalErr := json.Unmarshal([]byte(str), &b)

				// if the json is Unmarshalable, means that it is not a reference
				if unmarshalErr == nil {
					attrType := ""
					instillFormat := ""
					switch b.(type) {
					case string:
						attrType = "string"
						instillFormat = "text"
					case float64:
						attrType = "number"
						instillFormat = "number"
					case bool:
						attrType = "bool"
						instillFormat = "bool"
					case nil:
						attrType = "null"
						instillFormat = "null"
					}
					if isArrayReference {
						m, err = structpb.NewValue(map[string]interface{}{
							"title":       v.GetStructValue().Fields["title"].GetStringValue(),
							"description": v.GetStructValue().Fields["description"].GetStringValue(),
							"type":        "array",
							"items": map[string]interface{}{
								"type":          attrType,
								"instillFormat": instillFormat,
							},
						})
					} else {
						m, err = structpb.NewValue(map[string]interface{}{
							"title":         v.GetStructValue().Fields["title"].GetStringValue(),
							"description":   v.GetStructValue().Fields["description"].GetStringValue(),
							"type":          attrType,
							"instillFormat": instillFormat,
						})
					}

				} else {
					compId := strings.Split(str, ".")[0]
					str = str[len(strings.Split(str, ".")[0]):]
					upstreamCompIdx := -1
					for compIdx := range compsOrigin {
						if compsOrigin[compIdx].Id == compId {
							upstreamCompIdx = compIdx
						}
					}

					if upstreamCompIdx != -1 {
						comp := proto.Clone(compsOrigin[upstreamCompIdx]).(*pipelinePB.Component)

						var walk *structpb.Value
						if strings.HasPrefix(comp.DefinitionName, "connector-definitions") {
							task := ""
							if parsedTask, ok := comp.GetConfiguration().Fields["task"]; ok {
								task = parsedTask.GetStringValue()
							}
							if task == "" {
								keys := make([]string, 0, len(comp.GetConnectorDefinition().Spec.OpenapiSpecifications.GetFields()))
								if len(keys) != 1 {
									return nil, fmt.Errorf("must specify a task")
								}
								task = keys[0]
							}

							if _, ok := comp.GetConnectorDefinition().Spec.OpenapiSpecifications.GetFields()[task]; !ok {
								return nil, fmt.Errorf("generate OpenAPI spec error")
							}
							walk = comp.GetConnectorDefinition().Spec.OpenapiSpecifications.GetFields()[task]

							splits := strings.Split(str, ".")

							if splits[1] == "output" {
								for _, key := range []string{"paths", "/execute", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
									walk = walk.GetStructValue().Fields[key]
								}
							} else if splits[1] == "input" {
								for _, key := range []string{"paths", "/execute", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
									walk = walk.GetStructValue().Fields[key]
								}
							} else {
								return nil, fmt.Errorf("generate OpenAPI spec error")
							}
							str = str[len(splits[1])+1:]

						} else if comp.DefinitionName == "operator-definitions/op-start" {

							walk = structpb.NewStructValue(openApiInput)

						} else if utils.IsOperatorDefinition(comp.DefinitionName) {

							task := ""
							if parsedTask, ok := comp.GetConfiguration().Fields["task"]; ok {
								task = parsedTask.GetStringValue()
							}
							if task == "" {
								keys := make([]string, 0, len(comp.GetOperatorDefinition().Spec.OpenapiSpecifications.GetFields()))
								if len(keys) != 1 {
									return nil, fmt.Errorf("must specify a task")
								}
								task = keys[0]
							}

							if _, ok := comp.GetOperatorDefinition().Spec.OpenapiSpecifications.GetFields()[task]; !ok {
								return nil, fmt.Errorf("generate OpenAPI spec error")
							}

							walk = comp.GetOperatorDefinition().Spec.OpenapiSpecifications.GetFields()[task]

							splits := strings.Split(str, ".")

							if splits[1] == "output" {
								for _, key := range []string{"paths", "/execute", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
									walk = walk.GetStructValue().Fields[key]
								}
							} else if splits[1] == "input" {
								for _, key := range []string{"paths", "/execute", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
									walk = walk.GetStructValue().Fields[key]
								}
							} else {
								return nil, fmt.Errorf("generate OpenAPI spec error")
							}
							str = str[len(splits[1])+1:]
						}

						for {
							if len(str) == 0 {
								break
							}

							splits := strings.Split(str, ".")
							curr := splits[1]

							if strings.Contains(curr, "[") && strings.Contains(curr, "]") {
								target := strings.Split(curr, "[")[0]
								if _, ok := walk.GetStructValue().Fields["properties"]; ok {
									if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; !ok {
										return nil, fmt.Errorf("openapi error")
									}
								} else {
									return nil, fmt.Errorf("openapi error")
								}
								walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target].GetStructValue().Fields["items"]
							} else {
								target := curr

								if _, ok := walk.GetStructValue().Fields["properties"]; ok {
									if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; !ok {
										return nil, fmt.Errorf("openapi error")
									}
								} else {
									return nil, fmt.Errorf("openapi error")
								}

								walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]

							}

							str = str[len(curr)+1:]
						}

						if isArrayReference {
							m, err = structpb.NewValue(map[string]interface{}{
								"title":       v.GetStructValue().Fields["title"].GetStringValue(),
								"description": v.GetStructValue().Fields["description"].GetStringValue(),
								"type":        "array",
							})
							m.GetStructValue().Fields["items"] = structpb.NewStructValue(walk.GetStructValue())

						} else {
							m = structpb.NewStructValue(walk.GetStructValue())

							if _, ok := v.GetStructValue().Fields["title"]; ok {
								m.GetStructValue().Fields["title"] = v.GetStructValue().Fields["title"]
							} else {
								m.GetStructValue().Fields["title"] = structpb.NewStringValue("")
							}
							if _, ok := v.GetStructValue().Fields["description"]; ok {
								m.GetStructValue().Fields["description"] = v.GetStructValue().Fields["description"]
							} else {
								m.GetStructValue().Fields["description"] = structpb.NewStringValue("")
							}
						}

					} else {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}

				}

			} else {
				m, err = structpb.NewValue(map[string]interface{}{
					"title":         v.GetStructValue().Fields["title"].GetStringValue(),
					"description":   v.GetStructValue().Fields["description"].GetStringValue(),
					"type":          "string",
					"instillFormat": "text",
				})
			}
		case float64:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":         v.GetStructValue().Fields["title"].GetStringValue(),
				"description":   v.GetStructValue().Fields["description"].GetStringValue(),
				"type":          "number",
				"instillFormat": "number",
			})
		case bool:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":         v.GetStructValue().Fields["title"].GetStringValue(),
				"description":   v.GetStructValue().Fields["description"].GetStringValue(),
				"type":          "boolean",
				"instillFormat": "boolean",
			})
		case structpb.NullValue:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":         v.GetStructValue().Fields["title"].GetStringValue(),
				"description":   v.GetStructValue().Fields["description"].GetStringValue(),
				"type":          "null",
				"instillFormat": "null",
			})
		}
		if err != nil {
			success = false
		} else {
			openApiOutput.Fields["properties"].GetStructValue().Fields[k] = m
		}

	}

	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openApiOutput)

	if success {
		return template, nil
	}
	return nil, fmt.Errorf("generate OpenAPI spec error")

}
