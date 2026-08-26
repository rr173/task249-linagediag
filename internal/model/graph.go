package model

// ColumnRef 用「表限定名 + 列名」唯一定位一个列节点。
type ColumnRef struct {
	Table  string
	Column string
}

// DependencyGraph 内存中的列级血缘有向图，用于变更传播与环路检测。
// 节点为 ColumnRef；边表示「上游列 → 下游列」的派生关系。
type DependencyGraph struct {
	edges map[ColumnRef][]ColumnRef
	nodes map[ColumnRef]bool
}

// NewDependencyGraph 构造空图。
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		edges: make(map[ColumnRef][]ColumnRef),
		nodes: make(map[ColumnRef]bool),
	}
}

// AddNode 注册一个列节点。
func (g *DependencyGraph) AddNode(ref ColumnRef) {
	g.nodes[ref] = true
}

// HasNode 判断某列节点是否在图中。
func (g *DependencyGraph) HasNode(ref ColumnRef) bool {
	return g.nodes[ref]
}

// AddEdge 增加一条有向边（自动补全两端节点）。
func (g *DependencyGraph) AddEdge(src, dst ColumnRef) {
	g.AddNode(src)
	g.AddNode(dst)
	g.edges[src] = append(g.edges[src], dst)
}

// Downstream 返回从 start 出发可达的全部下游列（不含 start 本身），按先深顺序。
func (g *DependencyGraph) Downstream(start ColumnRef) []ColumnRef {
	seen := map[ColumnRef]bool{start: true}
	var order []ColumnRef
	stack := []ColumnRef{start}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, nxt := range g.edges[cur] {
			if !seen[nxt] {
				seen[nxt] = true
				order = append(order, nxt)
			}
		}
	}
	return order
}

// HasCycle 通过三色 DFS 判断全图是否存在环。
func (g *DependencyGraph) HasCycle() bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[ColumnRef]int)

	var dfs func(ColumnRef) bool
	dfs = func(u ColumnRef) bool {
		color[u] = gray
		for _, v := range g.edges[u] {
			if color[v] == gray {
				return true
			}
			if color[v] == white && dfs(v) {
				return true
			}
		}
		color[u] = black
		return false
	}

	for n := range g.nodes {
		if color[n] == white {
			if dfs(n) {
				return true
			}
		}
	}
	return false
}
