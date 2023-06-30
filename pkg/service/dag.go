package service

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
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
	comps            []*datamodel.Component
	compsIdx         map[string]int
	prerequisitesMap map[*datamodel.Component][]*datamodel.Component
	uf               *unionFind
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
	}
}

func (d *dag) AddEdge(from *datamodel.Component, to *datamodel.Component) {
	d.prerequisitesMap[from] = append(d.prerequisitesMap[from], to)
	d.uf.Union(d.compsIdx[from.Id], d.compsIdx[to.Id])
}

func (d *dag) TopoloicalSort() ([]*datamodel.Component, error) {
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

	if d.uf.Count() > 1 {
		return nil, fmt.Errorf("more then a graph")
	}
	if len(ans) < len(d.comps) {
		return nil, fmt.Errorf("not a valid dag")
	}
	return ans, nil
}
