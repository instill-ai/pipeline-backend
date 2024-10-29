package collection

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type appendInput struct {
	Array   []format.Value `key:"array"`
	Element format.Value   `key:"element"`
}

type appendOutput struct {
	Array []format.Value `key:"array"`
}

type assignInput struct {
	Data format.Value `key:"data"`
}

type assignOutput struct {
	Data format.Value `key:"data"`
}

type concatInput struct {
	Arrays [][]format.Value `key:"arrays"`
}

type concatOutput struct {
	Array []format.Value `key:"array"`
}

type differenceInput struct {
	SetA []format.Value `key:"set-a"`
	SetB []format.Value `key:"set-b"`
}

type differenceOutput struct {
	Set []format.Value `key:"set"`
}

type intersectionInput struct {
	Sets [][]format.Value `key:"sets"`
}

type intersectionOutput struct {
	Set []format.Value `key:"set"`
}

type splitInput struct {
	Array     []format.Value `key:"array"`
	GroupSize int            `key:"group-size"`
}

type splitOutput struct {
	Arrays [][]format.Value `key:"arrays"`
}

type unionInput struct {
	Sets [][]format.Value `key:"sets"`
}

type unionOutput struct {
	Set []format.Value `key:"set"`
}
