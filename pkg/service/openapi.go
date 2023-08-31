package service

import (
	"encoding/json"
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
func (s *service) GenerateOpenApiSpec(startComp *pipelinePB.Component, endComp *pipelinePB.Component, comps []*pipelinePB.Component) (*structpb.Struct, error) {
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
			switch t {
			case "integer_array", "number_array", "boolean_array":
				arrType = strings.Split(t, "_")[0]
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

		var err error

		switch v.GetStructValue().Fields["value"].AsInterface().(type) {
		case string:
			str := v.GetStructValue().Fields["value"].GetStringValue()
			if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") && !strings.HasPrefix(str, "{{") && !strings.HasSuffix(str, "}}") {
				// TODO
				str = str[1:]
				str = str[:len(str)-1]
				str = strings.ReplaceAll(str, " ", "")
				var b interface{}
				unmarshalErr := json.Unmarshal([]byte(str), &b)

				// if the json is Unmarshalable, means that it is not a reference
				if unmarshalErr == nil {
					attrType := ""
					switch b.(type) {
					case string:
						attrType = "string"
					case float64:
						attrType = "number"
					case bool:
						attrType = "bool"
					case nil:
						attrType = "null"
					}
					m, err = structpb.NewValue(map[string]interface{}{
						"title":       v.GetStructValue().Fields["title"].GetStringValue(),
						"description": v.GetStructValue().Fields["description"].GetStringValue(),
						"type":        attrType,
					})
				} else {
					compId := strings.Split(str, ".")[0]
					str = str[len(strings.Split(str, ".")[0]):]
					for compIdx := range comps {
						if comps[compIdx].Id == compId {
							if strings.HasPrefix(comps[compIdx].DefinitionName, "connector-definitions") {
								task := comps[compIdx].GetConfiguration().Fields["task"].GetStringValue()
								walk = comps[compIdx].GetConnectorDefinition().Spec.OpenapiSpecifications.GetFields()[task]
								for _, key := range []string{"paths", "/execute", "post", "responses", "200", "content", "application/json", "schema", "properties", "outputs", "items"} {
									walk = walk.GetStructValue().Fields[key]
								}

								for {
									splits := strings.Split(str, ".")
									if len(str) == 0 {
										break
									}

									curr := splits[1]

									if strings.Contains(curr, "[") && strings.Contains(curr, "]") {
										walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[strings.Split(curr, "[")[0]].GetStructValue().Fields["items"]
									} else {
										walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[curr]

									}

									str = str[len(curr)+1:]
								}
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

							if comps[compIdx].DefinitionName == "operator-definitions/start-operator" {

								isFullBody := str == ".body"
								str := str[len(strings.Split(str, ".")[1])+1:]

								walk = comps[compIdx].GetConfiguration().Fields["body"]
								for {

									splits := strings.Split(str, ".")
									if len(str) == 0 {
										break
									}

									curr := splits[1]

									if strings.Contains(curr, "[") && strings.Contains(curr, "]") {
										walk = walk.GetStructValue().Fields[strings.Split(curr, "[")[0]]
									} else {
										walk = walk.GetStructValue().Fields[curr]
									}

									str = str[len(curr)+1:]
								}

								if isFullBody {
									props := structpb.Struct{Fields: make(map[string]*structpb.Value)}
									// props := structpb.NewStructValue(walk.GetStructValue())
									for bodyK, bodyV := range walk.GetStructValue().Fields {
										attrType := ""
										arrType := ""
										switch t := bodyV.GetStructValue().Fields["type"].GetStringValue(); t {
										case "integer", "number", "boolean":
											attrType = t
										case "text", "image", "audio", "video":
											attrType = "string"
										default:
											attrType = "array"
											switch t {
											case "integer_array", "number_array", "boolean_array":
												arrType = strings.Split(t, "_")[0]
											case "text_array", "image_array", "audio_array", "video_array":
												arrType = "string"
											}
										}
										if attrType != "array" {

											props.Fields[bodyK], err = structpb.NewValue(map[string]interface{}{
												"title":       v.GetStructValue().Fields["title"].GetStringValue(),
												"description": v.GetStructValue().Fields["description"].GetStringValue(),
												"type":        attrType,
											})
											if err != nil {
												success = false
											}
										} else {
											props.Fields[bodyK], err = structpb.NewValue(map[string]interface{}{
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
									}
									m, err = structpb.NewValue(map[string]interface{}{
										"title":       v.GetStructValue().Fields["title"].GetStringValue(),
										"description": v.GetStructValue().Fields["description"].GetStringValue(),
										"type":        "object",
									})

									if err != nil {
										success = false
									}
									m.GetStructValue().Fields["properties"] = structpb.NewStructValue(&props)
								} else {
									attrType := ""
									arrType := ""
									switch t := walk.GetStructValue().Fields["type"].GetStringValue(); t {
									case "integer", "number", "boolean":
										attrType = t
									case "text", "image", "audio", "video":
										attrType = "string"
									default:
										attrType = "array"
										switch t {
										case "integer_array", "number_array", "boolean_array":
											arrType = strings.Split(t, "_")[0]
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
								}

							}

						}
					}

				}

			} else {
				m, err = structpb.NewValue(map[string]interface{}{
					"title":       v.GetStructValue().Fields["title"].GetStringValue(),
					"description": v.GetStructValue().Fields["description"].GetStringValue(),
					"type":        "string",
				})
			}
		case float64:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":       v.GetStructValue().Fields["title"].GetStringValue(),
				"description": v.GetStructValue().Fields["description"].GetStringValue(),
				"type":        "number",
			})
		case bool:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":       v.GetStructValue().Fields["title"].GetStringValue(),
				"description": v.GetStructValue().Fields["description"].GetStringValue(),
				"type":        "boolean",
			})
		case structpb.NullValue:
			m, err = structpb.NewValue(map[string]interface{}{
				"title":       v.GetStructValue().Fields["title"].GetStringValue(),
				"description": v.GetStructValue().Fields["description"].GetStringValue(),
				"type":        "null",
			})
		}
		if err != nil {
			success = false
		} else {
			openApiOutput.Fields["properties"].GetStructValue().Fields[k] = m
		}

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
