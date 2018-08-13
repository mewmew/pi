package cfg

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	golden := []struct {
		path string
	}{
		{path: "testdata/a.dot"},
	}
	for _, gold := range golden {
		buf, err := ioutil.ReadFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to read file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		g, err := ParseString(want)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		got := g.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestCopy(t *testing.T) {
	golden := []struct {
		path string
	}{
		{path: "testdata/a.dot"},
	}
	for _, gold := range golden {
		buf, err := ioutil.ReadFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to read file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		src, err := ParseString(want)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		dst := NewGraph()
		Copy(dst, src)
		got := dst.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestMerge(t *testing.T) {
	golden := []struct {
		path     string
		wantPath string
		nodes    map[string]bool
		id       string
	}{
		{
			path:     "testdata/sample.dot",
			wantPath: "testdata/sample.dot.I1.golden",
			nodes:    map[string]bool{"B1": true, "B2": true, "B3": true, "B4": true, "B5": true},
			id:       "I1",
		},
		{
			path:     "testdata/sample.dot",
			wantPath: "testdata/sample.dot.I3.golden",
			nodes:    map[string]bool{"B13": true, "B14": true, "B15": true},
			id:       "I3",
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Parse golden output.
		buf, err := ioutil.ReadFile(gold.wantPath)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		// Merge.
		out := Merge(in, gold.nodes, gold.id)
		got := out.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestInitDFSOrder(t *testing.T) {
	golden := []struct {
		path string
		want map[string]int
	}{
		{
			// Sample and reverse post-ordering taken from Fig. 2 in C. Cifuentes'
			// Structuring decompiled graphs [1].
			//
			// [1]: https://pdfs.semanticscholar.org/48bf/d31773af7b67f9d1b003b8b8ac889f08271f.pdf
			path: "testdata/sample.dot",
			want: map[string]int{
				"B1":  0,
				"B2":  1,
				"B3":  3,
				"B4":  2,
				"B5":  4,
				"B6":  5,
				"B7":  10,
				"B8":  11,
				"B9":  12,
				"B10": 13,
				"B11": 14,
				"B12": 6,
				"B13": 7,
				"B14": 8,
				"B15": 9,
			},
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Init pre- and post depth first search order.
		InitDFSOrder(in)
		// Check results.
		got := make(map[string]int)
		for _, n := range in.Nodes() {
			nn, ok := n.(*Node)
			if !ok {
				panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
			}
			// Compute reverse post-ordering.
			got[nn.name] = nn.RevPost
		}
		if !reflect.DeepEqual(got, gold.want) {
			t.Errorf("%q; output mismatch; expected `%v`, got `%v`", gold.path, gold.want, got)
			continue
		}
	}
}

func TestSortByRevPost(t *testing.T) {
	golden := []struct {
		path string
		want []string
	}{
		{
			// Sample and reverse post-ordering taken from Fig. 2 in C. Cifuentes'
			// Structuring decompiled graphs [1].
			//
			// [1]: https://pdfs.semanticscholar.org/48bf/d31773af7b67f9d1b003b8b8ac889f08271f.pdf
			path: "testdata/sample.dot",
			want: []string{"B1", "B2", "B4", "B3", "B5", "B6", "B12", "B13", "B14", "B15", "B7", "B8", "B9", "B10", "B11"},
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Init pre- and post depth first search order.
		InitDFSOrder(in)
		// Check results.
		var got []string
		for _, n := range SortByRevPost(in.Nodes()) {
			got = append(got, n.name)
		}
		if !reflect.DeepEqual(got, gold.want) {
			t.Errorf("%q; output mismatch; expected `%v`, got `%v`", gold.path, gold.want, got)
			continue
		}
	}
}
