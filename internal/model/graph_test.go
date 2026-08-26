package model

import "testing"

// TestHasCycleRegression 锁定「列变换形成循环依赖时必须拒绝构建」的修复。
// 修复前 HasCycle 的三色 DFS 存在死代码：第二个 if 条件恒假，递归从未深入 white 节点，
// 导致除自环外的环（两节点及以上）一律漏检，BuildLineage 误判成功、诊断结论不可信。
func TestHasCycleRegression(t *testing.T) {
	a := ColumnRef{Table: "db.s.t1", Column: "a"}
	b := ColumnRef{Table: "db.s.t2", Column: "b"}
	c := ColumnRef{Table: "db.s.t3", Column: "c"}

	t.Run("two_node_cycle", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEdge(a, b)
		g.AddEdge(b, a) // A→B→A
		if !g.HasCycle() {
			t.Fatal("two-node cycle must be detected")
		}
	})

	t.Run("three_node_cycle", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEdge(a, b)
		g.AddEdge(b, c)
		g.AddEdge(c, a) // A→B→C→A
		if !g.HasCycle() {
			t.Fatal("three-node cycle must be detected")
		}
	})

	t.Run("self_loop", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEdge(a, a) // A→A
		if !g.HasCycle() {
			t.Fatal("self-loop must be detected")
		}
	})

	t.Run("disconnected_cycle", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEdge(a, b) // 无环节点对
		x := ColumnRef{Table: "db.s.tx", Column: "x"}
		y := ColumnRef{Table: "db.s.ty", Column: "y"}
		g.AddEdge(x, y)
		g.AddEdge(y, x) // 在另一连通分量上的环
		if !g.HasCycle() {
			t.Fatal("cycle in a non-starting component must be detected")
		}
	})

	t.Run("acyclic_chain", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEdge(a, b)
		g.AddEdge(b, c)
		if g.HasCycle() {
			t.Fatal("acyclic chain must not be flagged as cyclic")
		}
	})

	t.Run("empty_and_single", func(t *testing.T) {
		if NewDependencyGraph().HasCycle() {
			t.Fatal("empty graph must not be cyclic")
		}
		g := NewDependencyGraph()
		g.AddNode(a)
		if g.HasCycle() {
			t.Fatal("single node without edges must not be cyclic")
		}
	})
}
