// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strconv"
	"strings"
)

type astGraph struct {
	out    strings.Builder
	nextID int
}

func ExportASTGraph(prog *Program) []byte {
	g := &astGraph{}
	g.out.WriteString("digraph FNL_AST {\n")
	g.out.WriteString("  graph [rankdir=TB];\n")
	g.out.WriteString("  node [shape=box, style=\"rounded\", fontname=\"Consolas\"];\n")
	g.out.WriteString("  edge [fontname=\"Consolas\"];\n\n")
	root := g.node("Program")
	for i, stmt := range prog.Statements {
		child := g.stmt(stmt)
		g.edge(root, child, fmt.Sprintf("stmt %d", i+1))
	}
	g.out.WriteString("}\n")
	return []byte(g.out.String())
}

func (g *astGraph) stmt(stmt Stmt) string {
	switch s := stmt.(type) {
	case *VarDecl:
		id := g.node(fmt.Sprintf("VarDecl\nname: %s\ntype: %s", s.Name, s.Type))
		g.edge(id, g.expr(s.Value), "value")
		return id
	case *Assign:
		id := g.node(fmt.Sprintf("Assign\nname: %s", s.Name))
		g.edge(id, g.expr(s.Value), "value")
		return id
	case *PrintStmt:
		name := "PrintStmt"
		if s.Newline {
			name = "PrintlnStmt"
		}
		id := g.node(name)
		g.edge(id, g.expr(s.Value), "value")
		return id
	case *ExitStmt:
		id := g.node("ExitStmt")
		g.edge(id, g.expr(s.Code), "code")
		return id
	case *BreakStmt:
		return g.node("BreakStmt")
	case *IfStmt:
		id := g.node("IfStmt")
		g.edge(id, g.expr(s.Cond), "condition")
		thenID := g.block("then", s.Then)
		g.edge(id, thenID, "then")
		for i, branch := range s.ElseIf {
			branchID := g.ifBranch(branch, i+1)
			g.edge(id, branchID, fmt.Sprintf("elseif %d", i+1))
		}
		if len(s.Else) > 0 {
			elseID := g.block("else", s.Else)
			g.edge(id, elseID, "else")
		}
		return id
	case *WhileStmt:
		id := g.node("WhileStmt")
		g.edge(id, g.expr(s.Cond), "condition")
		g.edge(id, g.block("body", s.Body), "body")
		return id
	default:
		return g.node("UnknownStmt")
	}
}

func (g *astGraph) ifBranch(branch IfBranch, index int) string {
	id := g.node(fmt.Sprintf("IfBranch\nelseif %d", index))
	g.edge(id, g.expr(branch.Cond), "condition")
	g.edge(id, g.block("body", branch.Body), "body")
	return id
}

func (g *astGraph) block(name string, stmts []Stmt) string {
	id := g.node(fmt.Sprintf("Block\n%s", name))
	for i, stmt := range stmts {
		child := g.stmt(stmt)
		g.edge(id, child, fmt.Sprintf("stmt %d", i+1))
	}
	return id
}

func (g *astGraph) expr(expr Expr) string {
	switch e := expr.(type) {
	case *BinaryExpr:
		id := g.node(fmt.Sprintf("BinaryExpr\nop: %s", tokenOp(e.Op)))
		g.edge(id, g.expr(e.Left), "left")
		g.edge(id, g.expr(e.Right), "right")
		return id
	case *UnaryExpr:
		id := g.node(fmt.Sprintf("UnaryExpr\nop: %s", tokenOp(e.Op)))
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *LiteralExpr:
		return g.node(fmt.Sprintf("LiteralExpr\ntype: %s\nvalue: %s", e.Type, e.Value))
	case *VarExpr:
		return g.node(fmt.Sprintf("VarExpr\nname: %s", e.Name))
	case *StrCallExpr:
		id := g.node("ToStrCallExpr")
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *InputCallExpr:
		return g.node("InputCallExpr")
	case *IsIntCallExpr:
		id := g.node("IsIntCallExpr")
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *ToIntCallExpr:
		id := g.node("ToIntCallExpr")
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *IsDoubleCallExpr:
		id := g.node("IsDoubleCallExpr")
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *ToDoubleCallExpr:
		id := g.node("ToDoubleCallExpr")
		g.edge(id, g.expr(e.Value), "value")
		return id
	case *MathRandomCallExpr:
		return g.node("MathRandomCallExpr")
	default:
		return g.node("UnknownExpr")
	}
}

func (g *astGraph) node(label string) string {
	id := fmt.Sprintf("n%d", g.nextID)
	g.nextID++
	g.out.WriteString("  ")
	g.out.WriteString(id)
	g.out.WriteString(" [label=")
	g.out.WriteString(strconv.Quote(label))
	g.out.WriteString("];\n")
	return id
}

func (g *astGraph) edge(from, to, label string) {
	g.out.WriteString("  ")
	g.out.WriteString(from)
	g.out.WriteString(" -> ")
	g.out.WriteString(to)
	if label != "" {
		g.out.WriteString(" [label=")
		g.out.WriteString(strconv.Quote(label))
		g.out.WriteString("]")
	}
	g.out.WriteString(";\n")
}
