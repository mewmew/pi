package cfg

import (
	"gonum.org/v1/gonum/graph"
)

// Merge returns a new control flow graph where the specified nodes have been
// collapsed into a single node with the new node name, and the predecessors and
// successors of the specified nodes.
func Merge(src *Graph, delNodes map[string]bool, newName string) *Graph {
	dst := NewGraph()
	Copy(dst, src)
	// preds marks predecessor nodes and records their edge attributes.
	preds := make(map[graph.Node]Attrs)
	succs := make(map[graph.Node]bool)
	newNode := dst.NewNodeWithName(newName)
	for delName := range delNodes {
		delNode := dst.nodeWithName(delName)
		if delNode.entry {
			newNode.entry = true
		}
		// Record predecessors not part of nodes.
		for _, pred := range dst.To(delNode.ID()) {
			p := node(pred)
			if !delNodes[p.name] {
				preds[dst.nodeWithName(p.name)] = edge(dst.Edge(p.ID(), delNode.ID())).Attrs
			}
		}
		// Record successors not part of nodes.
		for _, succ := range dst.From(delNode.ID()) {
			s := node(succ)
			if !delNodes[s.name] {
				succs[dst.nodeWithName(s.name)] = true
			}
		}
		dst.RemoveNode(delNode)
	}
	// Add new node after removing old nodes, to prevent potential collision with
	// previous entry node.
	dst.AddNode(newNode)
	// Add edges from predecessors to new node.
	for pred, attrs := range preds {
		e := edge(dst.NewEdge(pred, newNode))
		e.Attrs = attrs
		dst.SetEdge(e)
	}
	// Add edges from new node to successors.
	for succ := range succs {
		e := dst.NewEdge(newNode, succ)
		dst.SetEdge(e)
	}
	return dst
}
