package recipe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go/ast"
	"go/token"

	"github.com/PaesslerAG/jsonpath"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	componentbase "github.com/instill-ai/component/base"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

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
	compMap          datamodel.ComponentMap
	compsIdx         map[string]int
	prerequisitesMap map[string][]string
	uf               *unionFind
	ancestorsMap     map[string][]string
}

func NewDAG(compMap datamodel.ComponentMap) *dag {

	uf := NewUnionFind(len(compMap))

	return &dag{
		compMap:          compMap,
		uf:               uf,
		prerequisitesMap: map[string][]string{},
		ancestorsMap:     map[string][]string{},
	}
}

func (d *dag) AddEdge(from string, to string) {
	d.prerequisitesMap[from] = append(d.prerequisitesMap[from], to)
	d.uf.Union(d.compsIdx[from], d.compsIdx[to])
	if d.ancestorsMap[to] == nil {
		d.ancestorsMap[to] = []string{}
	}
	d.ancestorsMap[to] = append(d.ancestorsMap[to], from)
	d.ancestorsMap[to] = append(d.ancestorsMap[to], d.ancestorsMap[from]...)
}

func (d *dag) GetUpstreamCompIDs(id string) []string {
	return d.ancestorsMap[id]
}

type topologicalSortNode struct {
	compID string
	group  int // the group order
}

// TopologicalSort returns the topological sorted components
// the result is a list of list of components
// each list is a group of components that can be executed in parallel
func (d *dag) TopologicalSort() ([]datamodel.ComponentMap, error) {

	if len(d.compMap) == 0 {
		return []datamodel.ComponentMap{}, nil
	}

	indegreesMap := map[string]int{}
	for _, tos := range d.prerequisitesMap {
		for _, to := range tos {
			indegreesMap[to]++
		}

	}
	q := []*topologicalSortNode{}
	for id := range d.compMap {
		if indegreesMap[id] == 0 {
			q = append(q, &topologicalSortNode{
				compID: id,
				group:  0,
			})
		}
	}

	ans := []datamodel.ComponentMap{}

	count := 0
	taken := make(map[string]bool)
	for len(q) > 0 {
		from := q[0]
		q = q[1:]
		if len(ans) <= from.group {
			ans = append(ans, datamodel.ComponentMap{})
		}
		ans[from.group][from.compID] = d.compMap[from.compID]
		count += 1
		taken[from.compID] = true

		for _, to := range d.prerequisitesMap[from.compID] {
			indegreesMap[to]--
			if indegreesMap[to] == 0 {
				q = append(q, &topologicalSortNode{
					compID: to,
					group:  from.group + 1,
				})
			}
		}
	}

	if count < len(d.compMap) {
		return nil, fmt.Errorf("not a valid dag")
	}

	return ans, nil
}

func splitFunc(s rune) bool {
	return s == '.' || s == '['
}
func traverseBinding(memory *Memory, path string) (any, error) {

	compsMemory := memory.Component
	varsMemory := memory.Variable
	secretsMemory := memory.Secret

	splits := strings.FieldsFunc(path, splitFunc)

	newPath := ""
	for _, split := range splits {
		if strings.HasSuffix(split, "]") {
			newPath += fmt.Sprintf("[%s", split)
		} else {
			newPath += fmt.Sprintf("[\"%s\"]", split)
		}
	}

	m := map[string]any{
		SegMemory: map[string]any{},
	}
	for k := range compsMemory {
		m[SegMemory].(map[string]any)[k] = compsMemory[k]
	}

	if varsMemory != nil {
		m[SegMemory].(map[string]any)[SegVariable] = varsMemory
	}
	if secretsMemory != nil {
		m[SegMemory].(map[string]any)[SegSecret] = secretsMemory
	}

	b, _ := json.Marshal(m)
	var mParsed any
	_ = json.Unmarshal(b, &mParsed)
	res, err := jsonpath.Get(fmt.Sprintf("$.%s%s", SegMemory, newPath), mParsed)
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

func RenderInput(inputTemplate any, dataIndex int, memory *Memory) (any, error) {

	switch input := inputTemplate.(type) {
	case string:
		if strings.HasPrefix(input, "${") && strings.HasSuffix(input, "}") && strings.Count(input, "${") == 1 {
			input = input[2:]
			input = input[:len(input)-1]
			input = strings.TrimSpace(input)
			if input == SegSecret+"."+constant.GlobalSecretKey {
				return componentbase.SecretKeyword, nil
			}

			val, err := traverseBinding(memory, input)
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
			v, err := traverseBinding(memory, ref)
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
			converted, err := RenderInput(v, dataIndex, memory)
			if err != nil {
				return "", err
			}
			val[k] = converted

		}
		return val, nil
	case []any:
		val := []any{}
		for _, v := range input {
			converted, err := RenderInput(v, dataIndex, memory)
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
		// Convert InputsMemory and ComponentItemMemory into map[string]any.
		// Ignore error handling here since all of them are JSON data.
		b, _ := json.Marshal(v)
		m := map[string]any{}
		_ = json.Unmarshal(b, &m)
		return m[e.Sel.String()], nil
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

	case *ast.IndexExpr:
		v, err := EvalCondition(e.X, value)
		if err != nil {
			return nil, err
		}
		switch idxVal := e.Index.(type) {
		case *ast.BasicLit:
			// handle arr[index]
			if idxVal.Kind == token.INT {
				index, err := strconv.Atoi(idxVal.Value)
				if err != nil {
					return nil, err
				}
				return v.([]any)[index], nil
			}
			// handle obj[key]
			if idxVal.Kind == token.STRING {
				// key: remove ""
				key := idxVal.Value[1 : len(idxVal.Value)-1]
				return v.(map[string]any)[key], nil
			}
		}

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

func GenerateDAG(componentMap datamodel.ComponentMap) (*dag, error) {

	componentIDMap := make(map[string]bool)

	for id := range componentMap {
		componentIDMap[id] = true
		switch componentMap[id].Type {
		case datamodel.Iterator:
			for nestedID := range componentMap[id].Component {
				componentIDMap[nestedID] = true
			}
		}
	}

	graph := NewDAG(componentMap)

	for id, component := range componentMap {

		parents := []string{}

		if component.Condition != "" {
			parents = append(parents, FindReferenceParent(component.Condition)...)
		}

		switch component.Type {
		default:
			template, _ := json.Marshal(component.Input)
			parents = append(parents, FindReferenceParent(string(template))...)
		case datamodel.Iterator:
			parents = append(parents, FindReferenceParent(component.Input.(string))...)
			nestedComponentIDs := []string{id}
			for nestedID := range component.Component {
				nestedComponentIDs = append(nestedComponentIDs, nestedID)
			}
			for _, nestedComponent := range component.Component {

				if nestedComponent.Condition != "" {
					nestedParent := FindReferenceParent(nestedComponent.Condition)
					for idx := range nestedParent {
						if !slices.Contains(nestedComponentIDs, nestedParent[idx]) {
							parents = append(parents, nestedParent[idx])
						}
					}
				}

				switch nestedComponent.Type {
				case "default":
				default:
					template, _ := json.Marshal(nestedComponent.Input)
					nestedParent := FindReferenceParent(string(template))
					for idx := range nestedParent {
						if !slices.Contains(nestedComponentIDs, nestedParent[idx]) {
							parents = append(parents, nestedParent[idx])
						}
					}
				}
			}
		}

		for _, upstreamID := range parents {
			if _, ok := componentIDMap[upstreamID]; ok {
				graph.AddEdge(upstreamID, id)
			}

		}
	}

	return graph, nil
}

func FindReferenceParent(input string) []string {
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
}

func GenerateTraces(comps datamodel.ComponentMap, memory []*Memory) (map[string]*pb.Trace, error) {
	trace := map[string]*pb.Trace{}

	batchSize := len(memory)

	for compID := range comps {

		inputs := make([]*structpb.Struct, batchSize)
		outputs := make([]*structpb.Struct, batchSize)
		traceStatuses := make([]pb.Trace_Status, batchSize)

		for dataIdx := range batchSize {
			m := memory[dataIdx].Component[compID]
			if m.Status.Completed {
				traceStatuses[dataIdx] = pb.Trace_STATUS_COMPLETED
			} else if m.Status.Skipped {
				traceStatuses[dataIdx] = pb.Trace_STATUS_SKIPPED
			} else {
				traceStatuses[dataIdx] = pb.Trace_STATUS_ERROR
			}

			if m.Input != nil {

				in, err := json.Marshal(m.Input)
				if err != nil {
					return nil, err
				}
				inputStruct := &structpb.Struct{}

				err = protojson.Unmarshal(in, inputStruct)
				if err != nil {
					return nil, err
				}
				inputs[dataIdx] = inputStruct
			}

			if m.Output != nil {
				out, err := json.Marshal(m.Output)
				if err != nil {
					return nil, err
				}
				outputStruct := &structpb.Struct{}
				err = protojson.Unmarshal(out, outputStruct)
				if err != nil {
					return nil, err
				}
				outputs[dataIdx] = outputStruct
			}
		}

		trace[compID] = &pb.Trace{
			Statuses: traceStatuses,
			Inputs:   inputs,
			Outputs:  outputs,
		}
	}

	return trace, nil
}
