package main

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type NestedObject struct {
	NestedText *data.String `key:"nested-text"`
}

type Test struct {
	Text    *data.String           `key:"text"`
	Texts   []*data.String         `key:"texts"`
	Image   *data.Image            `key:"image"`
	Images  []*data.Image          `key:"images"`
	Object  NestedObject           `key:"object"`
	TextMap map[string]value.Value `key:"text-map"`
}
type Test2 struct {
	Text    *data.String           `key:"text"`
	Texts   []*data.String         `key:"texts"`
	Image   *data.Image            `key:"image"`
	Images  []*data.Image          `key:"images"`
	Object  NestedObject           `key:"object"`
	TextMap map[string]value.Value `key:"text-map"`
}

func main() {
	// Example struct
	img, _ := data.NewImageFromBytes([]byte{0xFF, 0xD8, 0xFF}, "", "")
	test := Test{
		Text:   data.NewString("example text"),
		Texts:  []*data.String{data.NewString("example text 1"), data.NewString("example text 2")},
		Image:  img,
		Images: []*data.Image{img, img},
		Object: NestedObject{
			data.NewString("example nested text"),
		},
		TextMap: map[string]value.Value{
			"tttt":  data.NewString("example text ddd"),
			"tttt1": data.NewString("example text dddd"),
			"tttt2": data.NewString("example text ddddd"),
		},
	}

	// fmt.Println(ConvertStructToValue(a))
	a, e := data.Marshal(test)
	fmt.Println(e)

	fmt.Println(a.ToStructValue())

	test2 := Test2{}
	e2 := data.Unmarshal(a, &test2)
	fmt.Println(e2)

	fmt.Println(test2)

	fmt.Println(test2.TextMap)
	fmt.Println(test2.Object.NestedText)
}
