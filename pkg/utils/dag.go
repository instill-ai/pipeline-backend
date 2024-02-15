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
		compsIdx[comps[idx].ID] = idx
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
	d.uf.Union(d.compsIdx[from.ID], d.compsIdx[to.ID])
	if d.ancestorsMap[to.ID] == nil {
		d.ancestorsMap[to.ID] = []string{}
	}
	d.ancestorsMap[to.ID] = append(d.ancestorsMap[to.ID], from.ID)
	d.ancestorsMap[to.ID] = append(d.ancestorsMap[to.ID], d.ancestorsMap[from.ID]...)
}

func (d *dag) GetAncestorIDs(id string) []string {
	return d.ancestorsMap[id]
}

type topologicalSortNode struct {
	comp  *datamodel.Component
	group int // the group order
}

// TopologicalSort returns the topological sorted components
// the result is a list of list of components
// each list is a group of components that can be executed in parallel
func (d *dag) TopologicalSort() ([][]*datamodel.Component, error) {

	if len(d.comps) == 0 {
		return nil, fmt.Errorf("no components")
	}

	indegreesMap := map[*datamodel.Component]int{}
	for _, tos := range d.prerequisitesMap {
		for _, to := range tos {
			indegreesMap[to]++
		}

	}
	q := []*topologicalSortNode{}
	for _, comp := range d.comps {
		if indegreesMap[comp] == 0 {
			q = append(q, &topologicalSortNode{
				comp:  comp,
				group: 0,
			})
		}
	}

	ans := [][]*datamodel.Component{}

	count := 0
	taken := make(map[*datamodel.Component]bool)
	for len(q) > 0 {
		from := q[0]
		q = q[1:]
		if len(ans) <= from.group {
			ans = append(ans, []*datamodel.Component{})
		}
		ans[from.group] = append(ans[from.group], from.comp)
		count += 1
		taken[from.comp] = true

		for _, to := range d.prerequisitesMap[from.comp] {
			indegreesMap[to]--
			if indegreesMap[to] == 0 {
				q = append(q, &topologicalSortNode{
					comp:  to,
					group: from.group + 1,
				})
			}
		}

	}

	if count < len(d.comps) {
		return nil, fmt.Errorf("not a valid dag")
	}

	if d.uf.Count() != 1 {
		return nil, fmt.Errorf("more than a dag")
	}

	return ans, nil
}

func traverseBinding(bindings any, path string) (any, error) {

	res, err := jsonpath.JsonPathLookup(bindings, "$."+path)
	if err != nil {
		// check primitive value
		var ret any
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
func RenderInput(input any, bindings map[string]any) (any, error) {

	switch input := input.(type) {
	case string:
		if strings.HasPrefix(input, "${") && strings.HasSuffix(input, "}") && strings.Count(input, "${") == 1 {
			input = input[2:]
			input = input[:len(input)-1]
			input = strings.TrimSpace(input)
			val, err := traverseBinding(bindings, input)
			if err != nil {
				return nil, err
			}
			return val, nil
		}

		val := ""
		for {
			startIdx := strings.Index(input, "${")
			if startIdx == -1 {
				val += input
				break
			}
			val += input[:startIdx]
			input = input[startIdx:]
			endIdx := strings.Index(input, "}")
			if endIdx == -1 {
				val += input
				break
			}

			ref := strings.TrimSpace(input[2:endIdx])
			v, err := traverseBinding(bindings, ref)
			if err != nil {
				return nil, err
			}

			switch v := v.(type) {
			case string:
				val += v
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				val += string(b)
			}
			input = input[endIdx+1:]
		}
		return val, nil

	case map[string]any:
		val := map[string]any{}
		for k, v := range input {
			converted, err := RenderInput(v, bindings)
			if err != nil {
				return "", err
			}
			val[k] = converted

		}
		return val, nil
	case []any:
		val := []any{}
		for _, v := range input {
			converted, err := RenderInput(v, bindings)
			if err != nil {
				return "", err
			}
			val = append(val, converted)
		}
		return val, nil
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

func EvalCondition(expr ast.Expr, value map[string]any) (any, error) {
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
		return v.(map[string]any)[e.Sel.String()], nil
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

func SanitizeCondition(cond string) (string, map[string]string, map[string]string) {
	varMapping := map[string]string{}
	revVarMapping := map[string]string{}
	varNameIdx := 0
	for {
		leftIdx := strings.Index(cond, "${")
		if leftIdx == -1 {
			break
		}
		rightIdx := strings.Index(cond, "}")

		left := cond[:leftIdx]
		v := cond[leftIdx+2 : rightIdx]
		right := cond[rightIdx+1:]

		srcName := strings.Split(strings.TrimSpace(v), ".")[0]
		if varName, ok := revVarMapping[srcName]; ok {
			varMapping[varName] = srcName
			revVarMapping[srcName] = varName
			cond = left + strings.ReplaceAll(v, srcName, varName) + right
		} else {
			varName := fmt.Sprintf("var%d", varNameIdx)
			varMapping[varName] = srcName
			revVarMapping[srcName] = varName
			varNameIdx++
			cond = left + strings.ReplaceAll(v, srcName, varName) + right
		}

	}

	return cond, varMapping, revVarMapping
}

func GenerateDAG(components []*datamodel.Component) (*dag, error) {
	componentIDMap := make(map[string]*datamodel.Component)

	for idx := range components {
		componentIDMap[components[idx].ID] = components[idx]
	}
	graph := NewDAG(components)
	for _, component := range components {

		configuration := proto.Clone(component.Configuration)
		template, _ := protojson.Marshal(configuration)

		condUpstreams := []string{}
		if cond := component.Configuration.Fields["condition"].GetStringValue(); cond != "" {
			var varMapping map[string]string
			cond, varMapping, _ = SanitizeCondition(cond)
			expr, err := parser.ParseExpr(cond)
			if err != nil {
				return nil, err
			}
			FindConditionUpstream(expr, &condUpstreams)

			for idx := range condUpstreams {

				if upstream, ok := componentIDMap[varMapping[condUpstreams[idx]]]; ok {
					graph.AddEdge(upstream, component)
				} else {
					return nil, fmt.Errorf("no condition upstream component '%s'", condUpstreams[idx])
				}

			}
		}

		parents := FindReferenceParent(string(template))
		for idx := range parents {
			if upstream, ok := componentIDMap[parents[idx]]; ok {
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
	var parsed any
	err := json.Unmarshal([]byte(input), &parsed)
	if err != nil {
		return []string{}
	}

	switch parsed := parsed.(type) {
	case string:

		upstreams := []string{}
		for {
			startIdx := strings.Index(input, "${")
			if startIdx == -1 {
				break
			}
			input = input[startIdx:]
			endIdx := strings.Index(input, "}")
			if endIdx == -1 {
				break
			}
			ref := strings.TrimSpace(input[2:endIdx])
			upstreams = append(upstreams, strings.Split(ref, ".")[0])
			input = input[endIdx+1:]
		}

		return upstreams

	case map[string]any:
		parents := []string{}
		for _, v := range parsed {
			encoded, err := json.Marshal(v)
			if err != nil {
				return []string{}
			}
			parents = append(parents, FindReferenceParent(string(encoded))...)

		}
		return parents
	case []any:
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
