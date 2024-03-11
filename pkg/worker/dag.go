package worker

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
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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

func traverseBinding(bindings ItemMemory, path string) (any, error) {

	// Note: We allow hyphens (-) in the component ID, but `jsonpath` doesn't support it.
	// This workaround transcodes the component ID into a valid name for the `jsonpath` library.
	componentID := strings.Split(path, ".")[0]
	route := strings.Join(strings.Split(path, ".")[1:], ".")
	componentIDMap := map[string]string{}
	transcodedBinding := map[string]any{}
	idx := 0
	for k := range bindings {
		// Generates a valid variable name for jsonpath
		newID := fmt.Sprintf("comp%d", idx)
		componentIDMap[k] = newID
		transcodedBinding[newID] = bindings[k]
		idx = idx + 1
	}
	res, err := jsonpath.Get(fmt.Sprintf("$.%s.%s", componentIDMap[componentID], route), transcodedBinding)
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
func RenderInput(input any, bindings ItemMemory) (any, error) {

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

func GenerateDAG(components []*datamodel.Component) (*dag, error) {
	componentIDMap := make(map[string]*datamodel.Component)

	var startComp *datamodel.Component
	for idx := range components {
		componentIDMap[components[idx].ID] = components[idx]
		if components[idx].IsStartComponent() {
			startComp = components[idx]
		}
		if components[idx].IsIteratorComponent() {
			for idx2 := range components[idx].IteratorComponent.Components {
				componentIDMap[components[idx].IteratorComponent.Components[idx2].ID] = components[idx].IteratorComponent.Components[idx2]
			}
		}
	}
	graph := NewDAG(components)

	// Force all component have start operator upstream
	if startComp != nil {
		for idx := range components {
			if !components[idx].IsStartComponent() {
				graph.AddEdge(startComp, components[idx])
			}
		}
	}

	for _, component := range components {

		parents := []string{}

		if component.IsConnectorComponent() || component.IsOperatorComponent() || component.IsIteratorComponent() {
			if component.GetCondition() != nil && *component.GetCondition() != "" {
				parents = append(parents, FindReferenceParent(*component.GetCondition())...)
			}

		}
		if component.IsConnectorComponent() {
			configuration := proto.Clone(component.ConnectorComponent.Input)
			template, _ := protojson.Marshal(configuration)
			parents = append(parents, FindReferenceParent(string(template))...)
		}
		if component.IsOperatorComponent() {
			configuration := proto.Clone(component.OperatorComponent.Input)
			template, _ := protojson.Marshal(configuration)
			parents = append(parents, FindReferenceParent(string(template))...)
		}
		if component.IsIteratorComponent() {
			parents = append(parents, FindReferenceParent(component.IteratorComponent.Input)...)
			nestedComponentIDs := []string{component.ID}
			for _, nestedComponent := range component.IteratorComponent.Components {
				nestedComponentIDs = append(nestedComponentIDs, nestedComponent.ID)
			}
			for _, nestedComponent := range component.IteratorComponent.Components {
				if nestedComponent.IsConnectorComponent() || nestedComponent.IsOperatorComponent() {
					if nestedComponent.GetCondition() != nil && *nestedComponent.GetCondition() != "" {
						nestedParent := FindReferenceParent(*nestedComponent.GetCondition())
						for idx := range nestedParent {
							if !slices.Contains(nestedComponentIDs, nestedParent[idx]) {
								parents = append(parents, nestedParent[idx])
							}
						}
					}
				}
				if nestedComponent.IsConnectorComponent() {
					configuration := proto.Clone(nestedComponent.ConnectorComponent.Input)
					template, _ := protojson.Marshal(configuration)
					nestedParent := FindReferenceParent(string(template))
					for idx := range nestedParent {
						if !slices.Contains(nestedComponentIDs, nestedParent[idx]) {
							parents = append(parents, nestedParent[idx])
						}
					}
				}
				if nestedComponent.IsOperatorComponent() {
					configuration := proto.Clone(nestedComponent.OperatorComponent.Input)
					template, _ := protojson.Marshal(configuration)
					nestedParent := FindReferenceParent(string(template))
					for idx := range nestedParent {
						if !slices.Contains(nestedComponentIDs, nestedParent[idx]) {
							parents = append(parents, nestedParent[idx])
						}
					}
				}
			}
		}
		if component.IsEndComponent() {
			for _, v := range component.EndComponent.Fields {
				parents = append(parents, FindReferenceParent(v.Value)...)
			}
		}

		for idx := range parents {
			if upstream, ok := componentIDMap[parents[idx]]; ok {
				graph.AddEdge(upstream, component)
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
func GenerateTraces(comps [][]*datamodel.Component, memory []ItemMemory, computeTime map[string]float32, batchSize int) (map[string]*pipelinePB.Trace, error) {
	trace := map[string]*pipelinePB.Trace{}
	for groupIdx := range comps {
		for compIdx := range comps[groupIdx] {
			inputs := []*structpb.Struct{}
			outputs := []*structpb.Struct{}
			var traceStatuses []pipelinePB.Trace_Status
			for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
				if checkComponentCompleted(memory[dataIdx][comps[groupIdx][compIdx].ID]) {
					traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_COMPLETED)
				} else if checkComponentSkipped(memory[dataIdx][comps[groupIdx][compIdx].ID]) {
					traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_SKIPPED)
				} else if checkComponentError(memory[dataIdx][comps[groupIdx][compIdx].ID]) {
					traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_ERROR)
				} else {
					traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_UNSPECIFIED)
				}

			}

			if !comps[groupIdx][compIdx].IsStartComponent() && !comps[groupIdx][compIdx].IsEndComponent() {
				for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
					if _, ok := memory[dataIdx][comps[groupIdx][compIdx].ID]["input"]; ok {
						data, err := json.Marshal(memory[dataIdx][comps[groupIdx][compIdx].ID]["input"])
						if err != nil {
							return nil, err
						}
						inputStruct := &structpb.Struct{}
						err = protojson.Unmarshal(data, inputStruct)
						if err != nil {
							return nil, err
						}
						inputs = append(inputs, inputStruct)
					}

				}
				for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
					if _, ok := memory[dataIdx][comps[groupIdx][compIdx].ID]["output"]; ok {
						data, err := json.Marshal(memory[dataIdx][comps[groupIdx][compIdx].ID]["output"])
						if err != nil {
							return nil, err
						}
						outputStruct := &structpb.Struct{}
						err = protojson.Unmarshal(data, outputStruct)
						if err != nil {
							return nil, err
						}
						outputs = append(outputs, outputStruct)
					}

				}
			}

			trace[comps[groupIdx][compIdx].ID] = &pipelinePB.Trace{
				Statuses:             traceStatuses,
				Inputs:               inputs,
				Outputs:              outputs,
				ComputeTimeInSeconds: computeTime[comps[groupIdx][compIdx].ID],
			}
		}
	}
	return trace, nil
}

func GenerateGlobalValue(pipelineUID uuid.UUID, recipe *datamodel.Recipe, ownerPermalink string) (map[string]any, error) {
	global := map[string]any{}

	global["pipeline"] = map[string]any{
		"uid":    pipelineUID.String(),
		"recipe": recipe,
	}
	global["owner"] = map[string]any{
		"uid": uuid.FromStringOrNil(strings.Split(ownerPermalink, "/")[1]),
	}

	return global, nil
}
