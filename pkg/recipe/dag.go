package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
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

func Render(ctx context.Context, template format.Value, batchIdx int, wfm memory.WorkflowMemory, allowUnresolved bool) (format.Value, error) {
	if input, ok := template.(format.ReferenceString); ok {
		s := input.String()
		if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") && strings.Count(s, "${") == 1 {
			s = s[2:]
			s = s[:len(s)-1]
			s = strings.TrimSpace(s)
			if s == constant.SegSecret+"."+constant.GlobalSecretKey {
				return data.NewString(componentbase.SecretKeyword), nil
			}
			val, err := wfm.Get(ctx, batchIdx, s)
			if err != nil {
				if allowUnresolved {
					return data.NewNull(), nil
				}
				return nil, errmsg.AddMessage(
					fmt.Errorf("resolving reference: %w", err),
					"Couldn't resolve reference "+s+".",
				)
			}
			return val, nil
		}

		val := ""
		for {
			startIdx := strings.Index(s, "${")
			if startIdx == -1 {
				val += s
				break
			}
			val += s[:startIdx]
			s = s[startIdx:]
			endIdx := strings.Index(s, "}")
			if endIdx == -1 {
				val += s
				break
			}

			ref := strings.TrimSpace(s[2:endIdx])
			v, err := wfm.Get(ctx, batchIdx, ref)
			if err != nil {
				if allowUnresolved {
					return data.NewNull(), nil
				}
				return nil, err
			}
			val += v.String()
			s = s[endIdx+1:]
		}
		return data.NewString(val), nil
	} else if input, ok := template.(data.Map); ok {
		var err error
		mp := data.Map{}
		for k, v := range input {
			if _, omittable := v.(format.OmittableField); !omittable {
				mp[k], err = Render(ctx, v, batchIdx, wfm, allowUnresolved)
				if err != nil {
					return nil, err
				}
			}

		}
		return mp, nil
	} else if input, ok := template.(data.Array); ok {
		var err error
		arr := make(data.Array, len(input))
		for i, v := range input {
			arr[i], err = Render(ctx, v, batchIdx, wfm, allowUnresolved)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	} else {
		return template, nil
	}
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
			if component.Input != nil {
				parents = append(parents, FindReferenceParent(component.Input.(string))...)
			}
			if component.Range != nil {
				switch rangeVal := component.Range.(type) {
				case []any:
					for _, v := range rangeVal {
						parents = append(parents, FindReferenceParent(fmt.Sprintf("%v", v))...)
					}
				case map[string]any:
					for _, v := range rangeVal {
						parents = append(parents, FindReferenceParent(fmt.Sprintf("%v", v))...)
					}
				}
			}
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

func GenerateTraces(ctx context.Context, wfm memory.WorkflowMemory, full bool) (map[string]*pb.Trace, error) {

	trace := map[string]*pb.Trace{}

	batchSize := wfm.GetBatchSize()

	for compID := range wfm.GetRecipe().Component {

		inputs := make([]*structpb.Struct, batchSize)
		outputs := make([]*structpb.Struct, batchSize)
		errors := make([]*structpb.Struct, batchSize)
		traceStatuses := make([]pb.Trace_Status, batchSize)

		for dataIdx := range batchSize {

			completed, err := wfm.GetComponentStatus(ctx, dataIdx, compID, memory.ComponentStatusCompleted)
			if err != nil {
				continue
			}
			skipped, err := wfm.GetComponentStatus(ctx, dataIdx, compID, memory.ComponentStatusSkipped)
			if err != nil {
				continue
			}
			if completed {
				traceStatuses[dataIdx] = pb.Trace_STATUS_COMPLETED
			} else if skipped {
				traceStatuses[dataIdx] = pb.Trace_STATUS_SKIPPED
			} else {
				traceStatuses[dataIdx] = pb.Trace_STATUS_ERROR
			}

			if compErr, err := wfm.GetComponentData(ctx, dataIdx, compID, memory.ComponentDataError); err == nil {
				structVal, err := compErr.ToStructValue()
				if err != nil {
					return nil, err
				}
				errors[dataIdx] = structVal.GetStructValue()
			}

			// TODO: For binary data fields, we should return a URL to access the blob instead of the raw data
			if full {
				if input, err := wfm.GetComponentData(ctx, dataIdx, compID, memory.ComponentDataInput); err == nil {
					structVal, err := input.ToStructValue()
					if err != nil {
						return nil, err
					}
					inputs[dataIdx] = structVal.GetStructValue()
				}

				if output, err := wfm.GetComponentData(ctx, dataIdx, compID, memory.ComponentDataOutput); err == nil {
					structVal, err := output.ToStructValue()
					if err != nil {
						return nil, err
					}
					outputs[dataIdx] = structVal.GetStructValue()
				}
			}
		}

		trace[compID] = &pb.Trace{
			Statuses: traceStatuses,
			Inputs:   inputs,
			Outputs:  outputs,

			// Note: Currently, all errors in a batch are the same.
			Error: errors[0],
		}
	}

	return trace, nil
}
