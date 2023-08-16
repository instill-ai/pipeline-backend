package operator

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

//go:embed start/config/definitions.json
var startOperatorDefinitionJson []byte

//go:embed end/config/definitions.json
var endOperatorDefinitionJson []byte

type Operator struct {
	// Store all the operator defintions
	definitionMapByUid map[uuid.UUID]*pipelinePB.OperatorDefinition
	definitionMapById  map[string]*pipelinePB.OperatorDefinition

	// Used for ordered
	definitionUids []uuid.UUID
}

func InitOperator() Operator {

	definitionMapByUid := map[uuid.UUID]*pipelinePB.OperatorDefinition{}
	definitionMapById := map[string]*pipelinePB.OperatorDefinition{}
	definitionUids := []uuid.UUID{}

	startOperatorDefinitions, _ := load(startOperatorDefinitionJson)
	for idx := range startOperatorDefinitions {
		definitionMapById[startOperatorDefinitions[idx].Id] = startOperatorDefinitions[idx]
		definitionMapByUid[uuid.FromStringOrNil(startOperatorDefinitions[idx].Uid)] = startOperatorDefinitions[idx]
		definitionUids = append(definitionUids, uuid.FromStringOrNil(startOperatorDefinitions[idx].Uid))
	}

	endOperatorDefinitions, _ := load(endOperatorDefinitionJson)
	for idx := range endOperatorDefinitions {
		definitionMapById[endOperatorDefinitions[idx].Id] = endOperatorDefinitions[idx]
		definitionMapByUid[uuid.FromStringOrNil(endOperatorDefinitions[idx].Uid)] = endOperatorDefinitions[idx]
		definitionUids = append(definitionUids, uuid.FromStringOrNil(endOperatorDefinitions[idx].Uid))
	}

	operator := Operator{
		definitionMapByUid: definitionMapByUid,
		definitionMapById:  definitionMapById,
		definitionUids:     definitionUids,
	}
	return operator
}

type definition struct {
	Custom           bool        `json:"custom"`
	DocumentationUrl string      `json:"documentation_url"`
	Icon             string      `json:"icon"`
	IconUrl          string      `json:"icon_url"`
	Id               string      `json:"id"`
	Public           bool        `json:"public"`
	Title            string      `json:"title"`
	Tombstone        bool        `json:"tombstone"`
	Uid              string      `json:"uid"`
	Spec             interface{} `json:"spec"`
}

func load(definitionsJson []byte) ([]*pipelinePB.OperatorDefinition, error) {
	defs := []*pipelinePB.OperatorDefinition{}
	var defJsonArr []definition
	err := json.Unmarshal(definitionsJson, &defJsonArr)
	if err != nil {
		panic(err)
	}

	for _, defJson := range defJsonArr {

		defJsonBytes, err := json.Marshal(defJson)
		if err != nil {
			return nil, err
		}

		def := &pipelinePB.OperatorDefinition{}

		err = protojson.Unmarshal(defJsonBytes, def)
		if err != nil {
			return nil, err
		}
		def.Name = fmt.Sprintf("operator-definitions/%s", def.Id)

		defs = append(defs, def)

	}
	return defs, nil
}

func (o *Operator) ListOperatorDefinitions() []*pipelinePB.OperatorDefinition {
	definitions := []*pipelinePB.OperatorDefinition{}
	for _, uid := range o.definitionUids {
		val, ok := o.definitionMapByUid[uid]
		if ok {
			definitions = append(definitions, val)
		}

	}

	return definitions
}

func (o *Operator) GetOperatorDefinitionByUid(defUid uuid.UUID) (*pipelinePB.OperatorDefinition, error) {
	val, ok := o.definitionMapByUid[defUid]
	if !ok {
		return nil, fmt.Errorf("get operator defintion error")
	}
	return val, nil
}

func (o *Operator) GetOperatorDefinitionById(defId string) (*pipelinePB.OperatorDefinition, error) {

	val, ok := o.definitionMapById[defId]
	if !ok {
		return nil, fmt.Errorf("get operator defintion error")
	}
	return val, nil
}
