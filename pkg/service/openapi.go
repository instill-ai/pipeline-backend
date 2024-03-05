package service

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const openAPISchemaTemplate = `{
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
func (s *service) GenerateOpenAPISpec(startCompOrigin *pipelinePB.Component, endCompOrigin *pipelinePB.Component, compsOrigin []*pipelinePB.Component) (*structpb.Struct, error) {
	success := true
	template := &structpb.Struct{}
	err := protojson.Unmarshal([]byte(openAPISchemaTemplate), template)
	if err != nil {
		return nil, err
	}

	var templateWalk *structpb.Value

	openAPIInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openAPIInput.Fields["type"] = structpb.NewStringValue("object")
	openAPIInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	startComp := proto.Clone(startCompOrigin).(*pipelinePB.Component)
	for k, v := range startComp.GetStartComponent().GetFields() {
		b, _ := protojson.Marshal(v)
		p := &structpb.Struct{}
		_ = protojson.Unmarshal(b, p)
		openAPIInput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(p)
	}

	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openAPIInput)
	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/triggerAsync", "post", "requestBody", "content", "application/json", "schema", "properties", "inputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openAPIInput)

	// output

	openAPIOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	openAPIOutput.Fields["type"] = structpb.NewStringValue("object")
	openAPIOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	endComp := proto.Clone(endCompOrigin).(*pipelinePB.Component)

	for k, v := range endComp.GetEndComponent().Fields {
		var m *structpb.Value

		var err error

		str := v.Value
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") && strings.Count(str, "${") == 1 {
			// TODO
			str = str[2:]
			str = str[:len(str)-1]
			str = strings.ReplaceAll(str, " ", "")

			compID := strings.Split(str, ".")[0]
			str = str[len(strings.Split(str, ".")[0]):]
			upstreamCompIdx := -1
			for compIdx := range compsOrigin {
				if compsOrigin[compIdx].Id == compID {
					upstreamCompIdx = compIdx
				}
			}

			if upstreamCompIdx != -1 {
				comp := proto.Clone(compsOrigin[upstreamCompIdx]).(*pipelinePB.Component)

				var walk *structpb.Value
				switch comp.Component.(type) {
				case *pipelinePB.Component_IteratorComponent:
					// TODO: implement this
					continue
				case *pipelinePB.Component_ConnectorComponent:
					task := comp.GetConnectorComponent().GetTask()
					if task == "" {
						keys := make([]string, 0, len(comp.GetConnectorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()))
						if len(keys) != 1 {
							return nil, fmt.Errorf("must specify a task")
						}
						task = keys[0]
					}

					if _, ok := comp.GetConnectorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()[task]; !ok {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}
					walk = comp.GetConnectorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()[task]

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
				case *pipelinePB.Component_StartComponent:
					walk = structpb.NewStructValue(openAPIInput)
				case *pipelinePB.Component_OperatorComponent:
					task := comp.GetOperatorComponent().GetTask()
					if task == "" {
						keys := make([]string, 0, len(comp.GetOperatorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()))
						if len(keys) != 1 {
							return nil, fmt.Errorf("must specify a task")
						}
						task = keys[0]
					}

					if _, ok := comp.GetOperatorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()[task]; !ok {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}

					walk = comp.GetOperatorComponent().GetDefinition().Spec.OpenapiSpecifications.GetFields()[task]

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
								break
							}
						} else {
							break
						}
						walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target].GetStructValue().Fields["items"]
					} else {
						target := curr

						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; !ok {
								break
							}
						} else {
							break
						}

						walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]

					}

					str = str[len(curr)+1:]
				}
				m = structpb.NewStructValue(walk.GetStructValue())

			} else {
				return nil, fmt.Errorf("generate OpenAPI spec error")
			}

			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["title"] = structpb.NewStringValue(v.Title)
			}
			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["description"] = structpb.NewStringValue(v.Description)
			}
			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["instillUIOrder"] = structpb.NewNumberValue(float64(v.InstillUiOrder))
			}

		} else {
			m, err = structpb.NewValue(map[string]interface{}{
				"title":          v.Title,
				"description":    v.Description,
				"instillUIOrder": v.InstillUiOrder,
				"type":           "string",
				"instillFormat":  "string",
			})
		}

		if err != nil {
			success = false
		} else {
			openAPIOutput.Fields["properties"].GetStructValue().Fields[k] = m
		}

	}

	templateWalk = template.GetFields()["paths"]
	for _, key := range []string{"/trigger", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
		templateWalk = templateWalk.GetStructValue().Fields[key]
	}
	*templateWalk = *structpb.NewStructValue(openAPIOutput)

	if success {
		return template, nil
	}
	return nil, fmt.Errorf("generate OpenAPI spec error")

}
