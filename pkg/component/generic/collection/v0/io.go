package collection

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type appendInput struct {
	Data  format.Value `instill:"data"`
	Value format.Value `instill:"value"`
}

type appendOutput struct {
	Data format.Value `instill:"data"`
}

type assignInput struct {
	Data  format.Value `instill:"data"`
	Path  string       `instill:"path"`
	Value format.Value `instill:"value"`
}

type assignOutput struct {
	Data format.Value `instill:"data"`
}

type concatInput struct {
	Data []format.Value `instill:"data"`
}

type concatOutput struct {
	Data format.Value `instill:"data"`
}

type differenceInput struct {
	Data []format.Value `instill:"data"`
}

type differenceOutput struct {
	Data format.Value `instill:"data"`
}

type intersectionInput struct {
	Data []format.Value `instill:"data"`
}

type intersectionOutput struct {
	Data format.Value `instill:"data"`
}

type splitInput struct {
	Data format.Value `instill:"data"`
	Size int          `instill:"size,default=1"`
}

type splitOutput struct {
	Data format.Value `instill:"data"`
}

type symmetricDifferenceInput struct {
	Data []format.Value `instill:"data"`
}

type symmetricDifferenceOutput struct {
	Data format.Value `instill:"data"`
}

type unionInput struct {
	Data []format.Value `instill:"data"`
}

type unionOutput struct {
	Data format.Value `instill:"data"`
}
