// Package cfg provides access to control flow graphs.
package cfg

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/graphism/simple"
	"github.com/llir/llvm/ir"
	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

var (
	// dbg represents a logger with the "cfg:" prefix, which logs debug messages
	// to standard error.
	dbg = log.New(os.Stderr, term.BlueBold("cfg:")+" ", 0)
	// warn represents a logger with the "cfg:" prefix, which logs warnings to
	// standard error.
	warn = log.New(os.Stderr, term.RedBold("cfg:")+" ", 0)
)

// === [ Graph ] ===============================================================

// Graph is a control flow graph.
type Graph struct {
	*simple.DirectedGraph
	// Graph ID.
	id string
	// Entry node of the control flow graph.
	entry graph.Node
	// nodes maps from node name to graph node.
	nodes map[string]*Node
}

// NewGraph returns a new control flow graph.
func NewGraph() *Graph {
	return &Graph{
		DirectedGraph: simple.NewDirectedGraph(),
		nodes:         make(map[string]*Node),
	}
}

// NewGraphFromFunc returns a new control flow graph based on the given
// function.
func NewGraphFromFunc(f *ir.Function) *Graph {
	g := NewGraph()
	// Force generate local IDs.
	_ = f.String()
	for i, block := range f.Blocks {
		from := nodeWithName(g, block.Name)
		if i == 0 {
			// Store entry node.
			g.SetEntry(from)
		}
		switch term := block.Term.(type) {
		case *ir.TermRet:
			// nothing to do.
		case *ir.TermBr:
			to := nodeWithName(g, term.Target.Name)
			edgeWithLabel(g, from, to, "")
		case *ir.TermCondBr:
			t := nodeWithName(g, term.TargetTrue.Name)
			f := nodeWithName(g, term.TargetFalse.Name)
			edgeWithLabel(g, from, t, "true")
			edgeWithLabel(g, from, f, "false")
		case *ir.TermSwitch:
			for _, c := range term.Cases {
				to := nodeWithName(g, c.Target.Name)
				label := fmt.Sprintf("case (x=%v)", c.X.Ident())
				edgeWithLabel(g, from, to, label)
			}
			to := nodeWithName(g, term.TargetDefault.Name)
			edgeWithLabel(g, from, to, "default case")
		case *ir.TermUnreachable:
			// nothing to do.
		default:
			panic(fmt.Errorf("support for terminator %T not yet implemented", term))
		}
	}
	return g
}

// nodeWithName returns the node of the given name. A new node is created if not
// yet present in the control flow graph.
func nodeWithName(g *Graph, name string) *Node {
	if n, ok := g.NodeWithName(name); ok {
		return n
	}
	n := g.NewNodeWithName(name)
	g.AddNode(n)
	return n
}

// edgeWithLabel adds a directed edge between the specified nodes and assignes
// it the given label.
func edgeWithLabel(g *Graph, from, to *Node, label string) *Edge {
	e := edge(g.NewEdge(from, to))
	if len(label) > 0 {
		e.Attrs["label"] = label
		switch label {
		case "true":
			e.Attrs["color"] = "darkgreen"
		case "false":
			e.Attrs["color"] = "red"
		}
	}
	g.SetEdge(e)
	return e
}

// String returns the string representation of the graph in Graphviz DOT format.
func (g *Graph) String() string {
	data, err := dot.Marshal(g, g.DOTID(), "", "\t", false)
	if err != nil {
		panic(fmt.Errorf("unable to marshal control flow graph in DOT format; %v", err))
	}
	return string(data)
}

// Entry returns the entry node of the control flow graph.
func (g *Graph) Entry() graph.Node {
	return g.entry
}

// SetEntry sets the entry node of the control flow graph.
func (g *Graph) SetEntry(n graph.Node) {
	nn := node(n)
	if g.entry != nil {
		panic(fmt.Errorf("cannot set %q as entry node; entry node %q already present", nn.DOTID(), node(g.entry).DOTID()))
	}
	nn.entry = true
	g.entry = nn
}

// NewNodeWithName returns a new node with the given name.
func (g *Graph) NewNodeWithName(name string) *Node {
	if len(name) == 0 {
		panic("empty node name")
	}
	n := g.NewNode()
	nn := node(n)
	nn.name = name
	return nn
}

// NodeWithName returns the node with the given name, and a boolean variable
// indicating success.
func (g *Graph) NodeWithName(name string) (*Node, bool) {
	n, ok := g.nodes[name]
	return n, ok
}

// TrueTarget returns the target node of the true branch from n.
func (g *Graph) TrueTarget(n *Node) *Node {
	succs := g.From(n.ID())
	if len(succs) != 2 {
		panic(fmt.Errorf("invalid number of successors; expected 2, got %d", len(succs)))
	}
	succ1 := node(succs[0])
	succ2 := node(succs[1])
	e1 := edge(g.Edge(n.ID(), succ1.ID()))
	e2 := edge(g.Edge(n.ID(), succ2.ID()))
	e1Label := e1.Attrs["label"]
	e2Label := e2.Attrs["label"]
	switch {
	case e1Label == "true" && e2Label == "false":
		return succ1
	case e1Label == "false" && e2Label == "true":
		return succ2
	default:
		// TODO: Figure out how to track edges of true- and false-branches in
		// between merges. For now, simply return the first successor (this will
		// lead to incorrect results, but at least lets us progress).
		warn.Printf(`unable to locate true branch of edges (%q -> %q) and (%q -> %q) based on edge label; expected "true" and "false", got %q and %q`, n.DOTID(), succ1.DOTID(), n.DOTID(), succ2.DOTID(), e1Label, e2Label)
		return succ1
	}
}

// FalseTarget returns the target node of the false branch from n.
func (g *Graph) FalseTarget(n *Node) *Node {
	succs := g.From(n.ID())
	if len(succs) != 2 {
		panic(fmt.Errorf("invalid number of successors; expected 2, got %d", len(succs)))
	}
	succ1 := node(succs[0])
	succ2 := node(succs[1])
	e1 := edge(g.Edge(n.ID(), succ1.ID()))
	e2 := edge(g.Edge(n.ID(), succ2.ID()))
	e1Label := e1.Attrs["label"]
	e2Label := e2.Attrs["label"]
	switch {
	case e1Label == "true" && e2Label == "false":
		return succ2
	case e1Label == "false" && e2Label == "true":
		return succ1
	default:
		// TODO: Figure out how to track edges of true- and false-branches in
		// between merges. For now, simply return the first successor (this will
		// lead to incorrect results, but at least lets us progress).
		warn.Printf(`unable to locate false branch of edges (%q -> %q) and (%q -> %q) based on edge label; expected "true" and "false", got %q and %q`, n.DOTID(), succ1.DOTID(), n.DOTID(), succ2.DOTID(), e1Label, e2Label)
		return succ1
	}
}

// initNodes initializes the mapping between node names and graph nodes.
func (g *Graph) initNodes() {
	for _, n := range g.Nodes() {
		nn := node(n)
		if len(nn.name) == 0 {
			panic(fmt.Errorf("invalid node; missing node name in %#v", nn))
		}
		if prev, ok := g.nodes[nn.name]; ok && nn != prev {
			panic(fmt.Errorf("node name %q already present in graph; prev node %#v, new node %#v", nn.name, prev, nn))
		}
		g.nodes[nn.name] = nn
	}
}

// --- [ dot.Graph ] -----------------------------------------------------------

// DOTID returns the DOT ID of the graph.
func (g *Graph) DOTID() string {
	return g.id
}

// --- [ dot.DOTIDSetter ] -----------------------------------------------------

// SetDOTID sets the DOT ID of the graph.
func (g *Graph) SetDOTID(id string) {
	g.id = id
}

// --- [ graph.NodeAdder ] -----------------------------------------------------

// NewNode returns a new node with a unique arbitrary ID.
func (g *Graph) NewNode() graph.Node {
	return &Node{
		Node:  g.DirectedGraph.NewNode(),
		Attrs: make(Attrs),
	}
}

// AddNode adds a node to the graph.
//
// If the added node ID matches an existing node ID, AddNode will panic.
func (g *Graph) AddNode(n graph.Node) {
	nn := node(n)
	g.DirectedGraph.AddNode(nn)
	if nn.entry {
		if g.entry != nil && nn != g.entry {
			panic(fmt.Errorf("entry node already set in graph; prev entry node %#v, new entry node %#v", g.entry, nn))
		}
		g.entry = nn
	}
	if len(nn.name) > 0 {
		if prev, ok := g.nodes[nn.name]; ok && nn != prev {
			panic(fmt.Errorf("node name %q already present in graph; prev node %#v, new node %#v", nn.name, prev, nn))
		}
		g.nodes[nn.name] = nn
	}
}

// --- [ graph.NodeRemover ] ---------------------------------------------------

// RemoveNode removes a node from the graph, as well as any edges attached to
// it. If the node is not in the graph it is a no-op.
func (g *Graph) RemoveNode(n graph.Node) {
	g.DirectedGraph.RemoveNode(n.ID())
	nn := node(n)
	delete(g.nodes, nn.name)
	if nn.entry {
		g.entry = nil
	}
}

// --- [ graph.EdgeAdder ] -----------------------------------------------------

// NewEdge returns a new edge from the source to the destination node.
func (g *Graph) NewEdge(from, to graph.Node) graph.Edge {
	return &Edge{
		Edge:  g.DirectedGraph.NewEdge(from, to),
		Attrs: make(Attrs),
	}
}

// SetEdge adds an edge from one node to another.
//
// If the graph supports node addition the nodes will be added if they do not
// exist, otherwise SetEdge will panic.
func (g *Graph) SetEdge(e graph.Edge) {
	ee, ok := e.(*Edge)
	if !ok {
		panic(fmt.Errorf("invalid edge type; expected *cfg.Edge, got %T", e))
	}
	// Add nodes if not yet present in graph.
	from, to := ee.From(), ee.To()
	if !g.Has(from.ID()) {
		g.AddNode(from)
	}
	if !g.Has(to.ID()) {
		g.AddNode(to)
	}
	// Add edge.
	g.DirectedGraph.SetEdge(ee)
}

// === [ Node ] ================================================================

// Node is a node in a control flow graph.
type Node struct {
	graph.Node
	// Node name (e.g. basic block label).
	name string
	// entry specifies whether the node is the entry node of the control flow
	// graph.
	entry bool
	// Depth first search preorder visit number.
	Pre int
	// Depth first search reverse postorder visit number.
	RevPost int
	// DOT attributes.
	Attrs

	// TODO: Figure out if we can move this information somewhere else; e.g.
	// local variables in loopStruct.

	// Number of back edges to the node.
	NBackEdges int
	// IsLatch specifies whether the node is a latch node.
	IsLatch bool
	// Type of the loop.
	LoopType LoopType
	// Header node of the loop.
	LoopHead *Node
	// Latch node of the loop.
	Latch *Node
	// Follow node of the loop.
	LoopFollow *Node
	// Follow node of the 2-way conditional.
	IfFollow *Node
	// Switch header node.
	SwitchHead *Node
	// Switch follow node.
	SwitchFollow *Node
}

//go:generate stringer -type LoopType -linecomment

// LoopType specifies the type of a loop.
type LoopType uint

// Loop types.
const (
	LoopTypeNone     LoopType = iota // none
	LoopTypePreTest                  // pre-test_loop
	LoopTypePostTest                 // post-test_loop
	LoopTypeEndless                  // endless_loop
)

// MarshalText encodes the loop type into UTF-8-encoded text and returns the
// result; implements encoding.TextMarshaler.
func (t LoopType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText decodes the loop type from the UTF-8 encoded text; implements
// encoding.TextUnmarshaler.
func (t *LoopType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "none":
		*t = LoopTypeNone
	case "pre-test_loop":
		*t = LoopTypePreTest
	case "post-test_loop":
		*t = LoopTypePostTest
	case "endless_loop":
		*t = LoopTypeEndless
	default:
		return errors.Errorf("support for unmarshalling loop type %q not yet implemented", s)
	}
	return nil
}

// --- [ dot.Node ] ------------------------------------------------------------

// DOTID returns the DOT ID of the node.
func (n *Node) DOTID() string {
	return n.name
}

// --- [ dot.DOTIDSetter ] -----------------------------------------------------

// SetDOTID sets the DOT ID of the node.
func (n *Node) SetDOTID(id string) {
	n.name = id
}

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of the node.
func (n *Node) Attributes() []encoding.Attribute {
	if n.entry {
		if prev, ok := n.Attrs["label"]; ok && prev != "entry" {
			panic(fmt.Errorf(`invalid DOT label of entry node; expected "entry", got %q`, prev))
		}
		n.Attrs["label"] = "entry"
	}
	return n.Attrs.Attributes()
}

// --- [ encoding.AttributeSetter ] -------------------------------------------

// SetAttribute sets the DOT attribute of the node.
func (n *Node) SetAttribute(attr encoding.Attribute) error {
	if attr.Key == "label" && attr.Value == "entry" {
		if prev, ok := n.Attrs["label"]; ok && prev != "entry" {
			panic(fmt.Errorf(`invalid DOT label of entry node; expected "entry", got %q`, prev))
		}
		n.entry = true
	} else {
		n.Attrs[attr.Key] = attr.Value
	}
	return nil
}

// === [ Edge ] ================================================================

// Edge is an edge in a control flow graph.
type Edge struct {
	graph.Edge
	// DOT attributes.
	Attrs
}

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of the edge.
func (e *Edge) Attributes() []encoding.Attribute {
	return e.Attrs.Attributes()
}

// --- [ encoding.AttributeSetter ] -------------------------------------------

// SetAttribute sets the DOT attribute of the edge.
func (e *Edge) SetAttribute(attr encoding.Attribute) error {
	e.Attrs[attr.Key] = attr.Value
	return nil
}

// ### [ Helper functions ] ####################################################

// Attrs specifies a set of DOT attributes as key-value pairs.
type Attrs map[string]string

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of a node or edge.
func (a Attrs) Attributes() []encoding.Attribute {
	var keys []string
	for key := range a {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var attrs []encoding.Attribute
	for _, key := range keys {
		attr := encoding.Attribute{
			Key:   key,
			Value: a[key],
		}
		// Quote label string if containing spaces.
		if key == "label" {
			s := attr.Value
			if strings.Contains(s, " ") && !strings.HasPrefix(s, `"`) {
				attr.Value = strconv.Quote(s)
			}
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

// node asserts that the given node is a control flow graph node.
func node(n graph.Node) *Node {
	if n, ok := n.(*Node); ok {
		return n
	}
	panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
}

// edge asserts that the given edge is a control flow graph edge.
func edge(e graph.Edge) *Edge {
	if e, ok := e.(*Edge); ok {
		return e
	}
	panic(fmt.Errorf("invalid edge type; expected *cfg.Edge, got %T", e))
}

// nodeWithName returns the node with the given name.
//
// If no matching node was located, nodeWithName panics.
func (g *Graph) nodeWithName(name string) *Node {
	n, ok := g.nodes[name]
	if !ok {
		panic(fmt.Errorf("unable to locate node with name %q", name))
	}
	return n
}
