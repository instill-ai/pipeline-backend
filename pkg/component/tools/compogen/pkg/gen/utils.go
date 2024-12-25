package gen

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func convertYAMLToJSON(yamlBytes []byte) ([]byte, error) {
	if yamlBytes == nil {
		return nil, nil
	}
	var d any
	err := yaml.Unmarshal(yamlBytes, &d)
	if err != nil {
		return nil, err
	}
	return json.Marshal(d)
}
