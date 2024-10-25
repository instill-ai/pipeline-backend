//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	_ "embed"

	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskAssign       = "TASK_ASSIGN"
	taskUnion        = "TASK_UNION"
	taskIntersection = "TASK_INTERSECTION"
	taskDifference   = "TASK_DIFFERENCE"
	taskAppend       = "TASK_APPEND"
	taskConcat       = "TASK_CONCAT"
	taskSplit        = "TASK_SPLIT"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
}

// Init returns an implementation of IOperator that processes JSON objects.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskAssign:
		e.execute = e.assign
	case taskUnion:
		e.execute = e.union
	case taskIntersection:
		e.execute = e.intersection
	case taskDifference:
		e.execute = e.difference
	case taskAppend:
		e.execute = e.append
	case taskConcat:
		e.execute = e.concat
	case taskSplit:
		e.execute = e.split
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

func (e *execution) split(in *structpb.Struct) (*structpb.Struct, error) {
	arr := in.Fields["array"].GetListValue().Values
	size := int(in.Fields["group-size"].GetNumberValue())
	groups := make([][]*structpb.Value, 0)

	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		groups = append(groups, arr[i:end])
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["arrays"] = structpb.NewListValue(&structpb.ListValue{Values: make([]*structpb.Value, len(groups))})

	for idx, g := range groups {
		out.Fields["arrays"].GetListValue().Values[idx] = structpb.NewListValue(&structpb.ListValue{Values: g})
	}

	return out, nil
}

func (e *execution) concat(in *structpb.Struct) (*structpb.Struct, error) {
	arrays := in.Fields["arrays"].GetListValue().Values
	concat := &structpb.ListValue{Values: []*structpb.Value{}}

	for _, a := range arrays {
		concat.Values = append(concat.Values, a.GetListValue().Values...)
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["array"] = structpb.NewListValue(concat)
	return out, nil
}

func (e *execution) union(in *structpb.Struct) (*structpb.Struct, error) {
	sets := in.Fields["sets"].GetListValue().Values
	cache := [][]string{}

	for _, s := range sets {
		c := []string{}
		for _, v := range s.GetListValue().Values {
			b, err := protojson.Marshal(v)
			if err != nil {
				return nil, err
			}
			c = append(c, string(b))
		}
		cache = append(cache, c)
	}

	set := &structpb.ListValue{Values: []*structpb.Value{}}
	un := lo.Union(cache...)
	for _, u := range un {
		var a any
		err := json.Unmarshal([]byte(u), &a)
		if err != nil {
			return nil, err
		}
		v, err := structpb.NewValue(a)
		if err != nil {
			return nil, err
		}
		set.Values = append(set.Values, v)
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["set"] = structpb.NewListValue(set)
	return out, nil
}

func (e *execution) intersection(in *structpb.Struct) (*structpb.Struct, error) {
	sets := in.Fields["sets"].GetListValue().Values

	if len(sets) == 1 {
		out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
		out.Fields["set"] = structpb.NewListValue(sets[0].GetListValue())
		return out, nil
	}

	curr := make([]string, len(sets[0].GetListValue().Values))
	for idx, v := range sets[0].GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		curr[idx] = string(b)
	}

	for _, s := range sets[1:] {
		next := make([]string, len(s.GetListValue().Values))
		for idx, v := range s.GetListValue().Values {
			b, err := protojson.Marshal(v)
			if err != nil {
				return nil, err
			}
			next[idx] = string(b)
		}

		i := lo.Intersect(curr, next)
		curr = i

	}

	set := &structpb.ListValue{Values: make([]*structpb.Value, len(curr))}

	for idx, c := range curr {
		var a any
		err := json.Unmarshal([]byte(c), &a)
		if err != nil {
			return nil, err
		}
		v, err := structpb.NewValue(a)
		if err != nil {
			return nil, err
		}
		set.Values[idx] = v
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["set"] = structpb.NewListValue(set)
	return out, nil
}

func (e *execution) difference(in *structpb.Struct) (*structpb.Struct, error) {
	setA := in.Fields["set-a"]
	setB := in.Fields["set-b"]

	valuesA := make([]string, len(setA.GetListValue().Values))
	for idx, v := range setA.GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		valuesA[idx] = string(b)
	}

	valuesB := make([]string, len(setB.GetListValue().Values))
	for idx, v := range setB.GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		valuesB[idx] = string(b)
	}
	dif, _ := lo.Difference(valuesA, valuesB)

	set := &structpb.ListValue{Values: make([]*structpb.Value, len(dif))}

	for idx, c := range dif {
		var a any

		err := json.Unmarshal([]byte(c), &a)
		if err != nil {
			return nil, err
		}
		v, err := structpb.NewValue(a)
		if err != nil {
			return nil, err
		}
		set.Values[idx] = v
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["set"] = structpb.NewListValue(set)
	return out, nil
}

func (e *execution) assign(in *structpb.Struct) (*structpb.Struct, error) {
	out := in
	return out, nil
}

func (e *execution) append(in *structpb.Struct) (*structpb.Struct, error) {
	arr := in.Fields["array"]
	element := in.Fields["element"]
	arr.GetListValue().Values = append(arr.GetListValue().Values, element)

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["array"] = arr
	return out, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
