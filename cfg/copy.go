package cfg

// Copy copies nodes and edges as directed edges from the source to the
// destination without first clearing the destination. Copy will panic if a node
// ID in the source graph matches a node ID in the destination.
func Copy(dst, src *Graph) {
	dst.id = src.id
	nodes := src.Nodes()
	for _, n := range nodes {
		dst.AddNode(n)
	}
	for _, u := range nodes {
		for _, v := range src.From(u.ID()) {
			dst.SetEdge(src.Edge(u.ID(), v.ID()))
		}
	}
	dst.initNodes()
}
