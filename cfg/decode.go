package cfg

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// Parse parses the given Graphviz DOT file into a control flow graph, reading
// from r.
func Parse(r io.Reader) (*Graph, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ParseBytes(buf)
}

// ParseFile parses the given Graphviz DOT file into a control flow graph,
// reading from path.
func ParseFile(path string) (*Graph, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ParseBytes(buf)
}

// ParseBytes parses the given Graphviz DOT file into a control flow graph,
// reading from b.
func ParseBytes(b []byte) (*Graph, error) {
	g := NewGraph()
	if err := dot.Unmarshal(b, g); err != nil {
		return nil, errors.WithStack(err)
	}
	// Initialize mapping between node names and graph nodes.
	g.initNodes()
	for _, n := range g.Nodes() {
		nn := node(n)
		if nn.entry {
			if g.entry != nil && nn != g.entry {
				panic(fmt.Errorf("entry node already set in graph; prev entry node %#v, new entry node %#v", g.entry, nn))
			}
			g.entry = nn
		}
	}
	if g.entry == nil {
		n, ok := g.NodeWithName(`"0"`)
		if !ok {
			panic(`unable to locate entry note or node with name "0"`)
		}
		g.SetEntry(n)
	}
	return g, nil
}

// ParseString parses the given Graphviz DOT file into a control flow graph,
// reading from s.
func ParseString(s string) (*Graph, error) {
	return ParseBytes([]byte(s))
}
