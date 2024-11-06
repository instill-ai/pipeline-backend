package collection

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type appendInput struct {
	Array   []format.Value `instill:"array"`
	Element format.Value   `instill:"element"`
}

type appendOutput struct {
	Array []format.Value `instill:"array"`
}

type assignInput struct {
	Data format.Value `instill:"data"`
}

type assignOutput struct {
	Data format.Value `instill:"data"`
}

type concatInput struct {
	Arrays [][]format.Value `instill:"arrays"`
}

type concatOutput struct {
	Array []format.Value `instill:"array"`
}

type differenceInput struct {
	SetA []format.Value `instill:"set-a"`
	SetB []format.Value `instill:"set-b"`
}

type differenceOutput struct {
	Set []format.Value `instill:"set"`
}

type intersectionInput struct {
	Sets [][]format.Value `instill:"sets"`
}

type intersectionOutput struct {
	Set []format.Value `instill:"set"`
}

type splitInput struct {
	Array     []format.Value `instill:"array"`
	GroupSize int            `instill:"group-size"`
}

type splitOutput struct {
	Arrays [][]format.Value `instill:"arrays"`
}

type unionInput struct {
	Sets [][]format.Value `instill:"sets"`
}

type unionOutput struct {
	Set []format.Value `instill:"set"`
}
