package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/oliveagle/jsonpath"
	"github.com/osteele/liquid"
	"github.com/osteele/liquid/render"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ComponentStatus struct {
	Started   bool
	Completed bool
	Skipped   bool
	Error     bool
}

type unionFind struct {
	roots []int
}

func NewUnionFind(size int) *unionFind {
	roots := []int{}
	for idx := 0; idx < size; idx++ {
		roots = append(roots, idx)
	}
	return &unionFind{
		roots: roots,
	}
}

func (uf *unionFind) find(x int) int {
	for x != uf.roots[x] {
		x = uf.roots[x]
	}
	return x
}

func (uf *unionFind) Union(x, y int) {
	rootX := uf.find(x)
	rootY := uf.find(y)
	if rootX != rootY {
		uf.roots[rootY] = rootX
	}
}
func (uf *unionFind) Count() int {
	count := 0
	for idx := range uf.roots {
		if idx == uf.roots[idx] {
			count++
		}
	}
	return count
}

type dag struct {
	comps            []*datamodel.Component
	compsIdx         map[string]int
	prerequisitesMap map[*datamodel.Component][]*datamodel.Component
	uf               *unionFind
	ancestorsMap     map[string][]string
}

func NewDAG(comps []*datamodel.Component) *dag {
	prerequisitesMap := map[*datamodel.Component][]*datamodel.Component{}
	compsIdx := map[string]int{}
	uf := NewUnionFind(len(comps))
	for idx := range comps {
		compsIdx[comps[idx].Id] = idx
	}

	return &dag{
		comps:            comps,
		compsIdx:         compsIdx,
		uf:               uf,
		prerequisitesMap: prerequisitesMap,
		ancestorsMap:     map[string][]string{},
	}
}

func (d *dag) AddEdge(from *datamodel.Component, to *datamodel.Component) {
	d.prerequisitesMap[from] = append(d.prerequisitesMap[from], to)
	d.uf.Union(d.compsIdx[from.Id], d.compsIdx[to.Id])
	if d.ancestorsMap[to.Id] == nil {
		d.ancestorsMap[to.Id] = []string{}
	}
	d.ancestorsMap[to.Id] = append(d.ancestorsMap[to.Id], from.Id)
	d.ancestorsMap[to.Id] = append(d.ancestorsMap[to.Id], d.ancestorsMap[from.Id]...)
}

func (d *dag) GetAncestorIDs(id string) []string {
	return d.ancestorsMap[id]
}

func (d *dag) TopologicalSort() ([]*datamodel.Component, error) {
	if len(d.comps) == 0 {
		return nil, fmt.Errorf("no components")
	}

	indegreesMap := map[*datamodel.Component]int{}
	for _, tos := range d.prerequisitesMap {
		for _, to := range tos {
			indegreesMap[to]++
		}

	}
	q := []*datamodel.Component{}
	for _, comp := range d.comps {
		if indegreesMap[comp] == 0 {
			q = append(q, comp)
		}
	}

	ans := []*datamodel.Component{}
	taken := make(map[*datamodel.Component]bool)
	for len(q) > 0 {
		from := q[0]
		q = q[1:]
		ans = append(ans, from)
		taken[from] = true
		for _, to := range d.prerequisitesMap[from] {
			indegreesMap[to]--
			if indegreesMap[to] == 0 {
				q = append(q, to)
			}
		}
	}

	if len(ans) < len(d.comps) {
		return nil, fmt.Errorf("not a valid dag")
	}

	if d.uf.Count() != 1 {
		return nil, fmt.Errorf("more than a dag")
	}

	return ans, nil
}

func traverseBinding(bindings interface{}, path string) (interface{}, error) {

	res, err := jsonpath.JsonPathLookup(bindings, "$."+path)
	if err != nil {
		// check primitive value
		var ret interface{}
		err := json.Unmarshal([]byte(path), &ret)
		if err != nil {
			return nil, fmt.Errorf("reference not correct: '%s'", path)
		}
		return ret, nil
	}
	switch res := res.(type) {
	default:
		return res, nil
	}
}
func RenderInput(input interface{}, bindings map[string]interface{}) (interface{}, error) {

	switch input := input.(type) {
	case string:
		if strings.HasPrefix(input, "${") && strings.HasSuffix(input, "}") && strings.Count(input, "${") == 1 {
			input = input[2:]
			input = input[:len(input)-1]
			input = strings.TrimSpace(input)
			if input[0] == '[' && input[len(input)-1] == ']' {
				outs := []interface{}{}
				subInputs := strings.Split(input[1:len(input)-1], ",")
				for _, subInput := range subInputs {
					out, err := traverseBinding(bindings, subInput)
					if err != nil {
						return nil, err
					}
					outs = append(outs, out)
				}
				return outs, nil

			} else {
				out, err := traverseBinding(bindings, input)
				if err != nil {
					return nil, err
				}
				return out, nil
			}
		}

		// TODO: we should retire Liquid instead of changing the delimiters
		engine := liquid.NewEngine().Delims("${", "}", "{%", "%}")
		out, err := engine.ParseAndRenderString(input, bindings)
		if err != nil {
			return nil, err
		}
		return out, err

	case map[string]interface{}:
		ret := map[string]interface{}{}
		for k, v := range input {
			converted, err := RenderInput(v, bindings)
			if err != nil {
				return "", err
			}
			ret[k] = converted

		}
		return ret, nil
	case []interface{}:
		ret := []interface{}{}
		for _, v := range input {
			converted, err := RenderInput(v, bindings)
			if err != nil {
				return "", err
			}
			ret = append(ret, converted)
		}
		return ret, nil
	default:
		return input, nil
	}
}

func FindConditionUpstream(expr ast.Expr, upstreams *[]string) {
	switch e := (expr).(type) {
	case *ast.BinaryExpr:
		FindConditionUpstream(e.X, upstreams)
		FindConditionUpstream(e.Y, upstreams)
	case *ast.ParenExpr:
		FindConditionUpstream(e.X, upstreams)
	case *ast.SelectorExpr:
		FindConditionUpstream(e.X, upstreams)
	case *ast.IndexExpr:
		FindConditionUpstream(e.X, upstreams)
	case *ast.Ident:
		if e.Name == "true" {
			return
		}
		if e.Name == "false" {
			return
		}
		*upstreams = append(*upstreams, e.Name)
	}
}

func EvalCondition(expr ast.Expr, value map[string]interface{}) (interface{}, error) {
	switch e := (expr).(type) {
	case *ast.UnaryExpr:
		xRes, err := EvalCondition(e.X, value)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case token.NOT: // !
			switch xVal := xRes.(type) {
			case bool:
				return !xVal, nil
			}
		case token.SUB: // -
			switch xVal := xRes.(type) {
			case int64:
				return -xVal, nil
			case float64:
				return -xVal, nil
			}
		}
	case *ast.BinaryExpr:

		xRes, err := EvalCondition(e.X, value)
		if err != nil {
			return nil, err
		}
		yRes, err := EvalCondition(e.Y, value)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case token.LAND: // &&

			xBool := false
			yBool := false
			switch xVal := xRes.(type) {
			case int64, float64:
				xBool = (xVal != 0)
			case string:
				xBool = (xVal != "")
			case bool:
				xBool = xVal
			}
			switch yVal := yRes.(type) {
			case int64, float64:
				yBool = (yVal != 0)
			case string:
				yBool = (yVal != "")
			case bool:
				yBool = yVal
			}
			return xBool && yBool, nil
		case token.LOR: // ||

			xBool := false
			yBool := false
			switch xVal := xRes.(type) {
			case int64, float64:
				xBool = (xVal != 0)
			case string:
				xBool = (xVal != "")
			case bool:
				xBool = xVal
			}
			switch yVal := yRes.(type) {
			case int64, float64:
				yBool = (yVal != 0)
			case string:
				yBool = (yVal != "")
			case bool:
				yBool = yVal
			}
			return xBool || yBool, nil

		case token.EQL: // ==
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal == yVal, nil
				case float64:
					return float64(xVal) == yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal == float64(yVal), nil
				case float64:
					return xVal == yVal, nil
				}
			}
			return reflect.DeepEqual(xRes, yRes), nil
		case token.NEQ: // !=
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal != yVal, nil
				case float64:
					return float64(xVal) != yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal != float64(yVal), nil
				case float64:
					return xVal != yVal, nil
				}
			}
			return !reflect.DeepEqual(xRes, yRes), nil

		case token.LSS: // <
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal < yVal, nil
				case float64:
					return float64(xVal) < yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal < float64(yVal), nil
				case float64:
					return xVal < yVal, nil
				}
			}
		case token.GTR: // >
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal > yVal, nil
				case float64:
					return float64(xVal) > yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal > float64(yVal), nil
				case float64:
					return xVal > yVal, nil
				}
			}

		case token.LEQ: // <=
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal <= yVal, nil
				case float64:
					return float64(xVal) <= yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal <= float64(yVal), nil
				case float64:
					return xVal <= yVal, nil
				}
			}
		case token.GEQ: // >=
			switch xVal := xRes.(type) {
			case int64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal >= yVal, nil
				case float64:
					return float64(xVal) >= yVal, nil
				}
			case float64:
				switch yVal := yRes.(type) {
				case int64:
					return xVal >= float64(yVal), nil
				case float64:
					return xVal >= yVal, nil
				}
			}

		}

	case *ast.ParenExpr:
		return EvalCondition(e.X, value)
	case *ast.SelectorExpr:
		v, err := EvalCondition(e.X, value)
		if err != nil {
			return nil, err
		}
		return v.(map[string]interface{})[e.Sel.String()], nil
	case *ast.BasicLit:
		if e.Kind == token.INT {
			return strconv.ParseInt(e.Value, 10, 64)
		}
		if e.Kind == token.FLOAT {
			return strconv.ParseFloat(e.Value, 64)
		}
		if e.Kind == token.STRING {
			return e.Value[1 : len(e.Value)-1], nil
		}
		return e.Value, nil
	case *ast.Ident:
		if e.Name == "true" {
			return true, nil
		}
		if e.Name == "false" {
			return false, nil
		}

		return value[e.Name], nil

	}
	return false, fmt.Errorf("condition error")
}

func GenerateDAG(components []*datamodel.Component) (*dag, error) {
	componentIdMap := make(map[string]*datamodel.Component)

	for idx := range components {
		componentIdMap[components[idx].Id] = components[idx]
	}
	graph := NewDAG(components)
	for _, component := range components {

		// TODO: we should retire Liquid instead of changing the delimiters
		engine := liquid.NewEngine().Delims("${", "}", "{%", "%}")
		configuration := proto.Clone(component.Configuration)
		template, _ := protojson.Marshal(configuration)
		out, err := engine.ParseTemplate(template)
		if err != nil {
			return nil, err
		}

		condUpstreams := []string{}
		if cond := component.Configuration.Fields["condition"].GetStringValue(); cond != "" {
			cond = strings.ReplaceAll(cond, "${", "")
			cond = strings.ReplaceAll(cond, "}", "")
			expr, err := parser.ParseExpr(cond)
			if err != nil {
				return nil, err
			}
			FindConditionUpstream(expr, &condUpstreams)
		}

		for idx := range condUpstreams {
			if upstream, ok := componentIdMap[condUpstreams[idx]]; ok {
				graph.AddEdge(upstream, component)
			} else {
				return nil, fmt.Errorf("no condition upstream component '%s'", condUpstreams[idx])
			}

		}

		for _, node := range out.GetRoot().(*render.SeqNode).Children {
			parents := []string{}
			switch node := node.(type) {
			case *render.ObjectNode:
				upstream := strings.Split(node.Args, ".")[0]
				parents = append(parents, upstream)
			}
			for idx := range parents {
				if _, ok := componentIdMap[parents[idx]]; ok {
					graph.AddEdge(componentIdMap[parents[idx]], component)
				}
			}
		}
		parents := FindReferenceParent(string(template))
		for idx := range parents {
			if upstream, ok := componentIdMap[parents[idx]]; ok {
				graph.AddEdge(upstream, component)
			} else {
				return nil, fmt.Errorf("no upstream component '%s'", parents[idx])
			}

		}
	}

	return graph, nil
}

// TODO: simplify this
func FindReferenceParent(input string) []string {
	var parsed interface{}
	err := json.Unmarshal([]byte(input), &parsed)
	if err != nil {
		return []string{}
	}

	switch parsed := parsed.(type) {
	case string:

		if strings.HasPrefix(parsed, "${") && strings.HasSuffix(parsed, "}") && strings.Count(parsed, "${") == 1 {

			parsed = parsed[2:]
			parsed = parsed[:len(parsed)-1]
			parsed = strings.TrimSpace(parsed)
			if parsed[0] == '[' && parsed[len(parsed)-1] == ']' {
				parents := []string{}
				subStrs := strings.Split(parsed[1:len(parsed)-1], ",")
				for _, subStr := range subStrs {
					var b interface{}
					err := json.Unmarshal([]byte(subStr), &b)

					// if the json is Unmarshalable, means that it is not a reference
					if err == nil {
						continue
					}
					parents = append(parents, strings.Split(subStr, ".")[0])
				}
				return parents

			} else {
				var b interface{}
				err := json.Unmarshal([]byte(parsed), &b)

				// if the json is Unmarshalable, means that it is not a reference
				if err == nil {
					return []string{}
				}
				return []string{strings.Split(parsed, ".")[0]}
			}
		}
		return []string{}

	case map[string]interface{}:
		parents := []string{}
		for _, v := range parsed {
			encoded, err := json.Marshal(v)
			if err != nil {
				return []string{}
			}
			parents = append(parents, FindReferenceParent(string(encoded))...)

		}
		return parents
	case []interface{}:
		parents := []string{}
		for _, v := range parsed {
			encoded, err := json.Marshal(v)
			if err != nil {
				return []string{}
			}
			parents = append(parents, FindReferenceParent(string(encoded))...)
			if err != nil {
				return []string{}
			}

		}
		return parents

	}

	return []string{}
}
