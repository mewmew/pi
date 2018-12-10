// TODO: remove this file in favour of gonum/graph alternatives/integration.

package cfg

import (
	"fmt"
	"sort"

	"bitbucket.org/zombiezen/cardcpx/natsort"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// InitDFSOrder initializes the pre- and post depth first search visit order of
// each node.
func InitDFSOrder(g *Graph) {
	visited := make(map[graph.Node]bool)
	// post-order
	var walk func(n graph.Node)
	first := 0
	last := g.Nodes().Len() - 1
	walk = func(n graph.Node) {
		nn, ok := n.(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; exepcted *cfg.Node, got %T", n))
		}
		nn.Pre = first
		first++
		visited[n] = true
		for _, succ := range sortByDOTID(graph.NodesOf(g.From(n.ID()))) {
			if !visited[succ] {
				walk(succ)
			}
		}
		nn.RevPost = last
		last--
	}
	walk(g.entry)
	// Ensure that all nodes have been visited.
	for _, n := range sortByDOTID(graph.NodesOf(g.Nodes())) {
		if !visited[n] {
			walk(n)
		}
	}
}

// SortByRevPost sorts the given list of nodes by reverse post-order.
func SortByRevPost(ns []graph.Node) []*Node {
	less := func(i, j int) bool {
		a, ok := ns[i].(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; exepcted *cfg.Node, got %T", ns[i]))
		}
		b, ok := ns[j].(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; exepcted *cfg.Node, got %T", ns[j]))
		}
		return a.RevPost < b.RevPost
	}
	sort.Slice(ns, less)
	var nodes []*Node
	for _, n := range ns {
		nodes = append(nodes, node(n))
	}
	return nodes
}

// SortByPost sorts the given list of nodes by post-order.
func SortByPost(ns []graph.Node) []*Node {
	less := func(i, j int) bool {
		a, ok := ns[i].(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; exepcted *cfg.Node, got %T", ns[i]))
		}
		b, ok := ns[j].(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; exepcted *cfg.Node, got %T", ns[j]))
		}
		return b.RevPost < a.RevPost
	}
	sort.Slice(ns, less)
	var nodes []*Node
	for _, n := range ns {
		nodes = append(nodes, node(n))
	}
	return nodes
}

// sortByDOTID sorts the given list of nodes by DOT ID if present, and node ID
// otherwise.
func sortByDOTID(ns []graph.Node) []graph.Node {
	dotIDs := true
	for _, n := range ns {
		_, ok := n.(dot.Node)
		if !ok {
			dotIDs = false
			break
		}
	}
	less := func(i, j int) bool {
		if dotIDs {
			a := ns[i].(dot.Node).DOTID()
			b := ns[j].(dot.Node).DOTID()
			return natsort.Less(a, b)
		}
		return ns[i].ID() < ns[j].ID()
	}
	sort.Slice(ns, less)
	return ns
}
