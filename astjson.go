// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"encoding/json"
	"fmt"
)

const (
	astJSONFormat  = "fnl.ast"
	astJSONVersion = 1
)

type astKind struct {
	Kind string `json:"kind"`
}

type astFile struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	Kind       string            `json:"kind"`
	Statements []json.RawMessage `json:"statements"`
}

type astOutputFile struct {
	Format     string `json:"format"`
	Version    int    `json:"version"`
	Kind       string `json:"kind"`
	Statements []any  `json:"statements"`
}

func ExportAST(prog *Program) ([]byte, error) {
	out := astOutputFile{
		Format:     astJSONFormat,
		Version:    astJSONVersion,
		Kind:       "Program",
		Statements: encodeStmtList(prog.Statements),
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func ImportAST(data []byte) (*Program, error) {
	var file astFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Format != astJSONFormat {
		return nil, fmt.Errorf("AST JSON format must be %q", astJSONFormat)
	}
	if file.Version != astJSONVersion {
		return nil, fmt.Errorf("unsupported AST JSON version %d", file.Version)
	}
	if file.Kind != "Program" {
		return nil, fmt.Errorf("AST root kind must be Program, got %q", file.Kind)
	}
	stmts, err := decodeStmtList(file.Statements)
	if err != nil {
		return nil, err
	}
	return &Program{Statements: stmts}, nil
}

func encodeStmtList(stmts []Stmt) []any {
	out := make([]any, 0, len(stmts))
	for _, stmt := range stmts {
		out = append(out, encodeStmt(stmt))
	}
	return out
}

func encodeStmt(stmt Stmt) any {
	switch s := stmt.(type) {
	case *VarDecl:
		return map[string]any{"kind": "VarDecl", "pos": s.Pos, "name": s.Name, "type": s.Type, "value": encodeExpr(s.Value)}
	case *Assign:
		return map[string]any{"kind": "Assign", "pos": s.Pos, "name": s.Name, "value": encodeExpr(s.Value)}
	case *PrintStmt:
		return map[string]any{"kind": "PrintStmt", "pos": s.Pos, "newline": s.Newline, "value": encodeExpr(s.Value)}
	case *ExitStmt:
		return map[string]any{"kind": "ExitStmt", "pos": s.Pos, "code": encodeExpr(s.Code)}
	case *BreakStmt:
		return map[string]any{"kind": "BreakStmt", "pos": s.Pos}
	case *IfStmt:
		return map[string]any{
			"kind":      "IfStmt",
			"pos":       s.Pos,
			"condition": encodeExpr(s.Cond),
			"then":      encodeStmtList(s.Then),
			"elseif":    encodeIfBranches(s.ElseIf),
			"else":      encodeStmtList(s.Else),
		}
	case *WhileStmt:
		return map[string]any{"kind": "WhileStmt", "pos": s.Pos, "condition": encodeExpr(s.Cond), "body": encodeStmtList(s.Body)}
	default:
		return map[string]any{"kind": "UnknownStmt"}
	}
}

func encodeIfBranches(branches []IfBranch) []any {
	out := make([]any, 0, len(branches))
	for _, branch := range branches {
		out = append(out, map[string]any{
			"kind":      "IfBranch",
			"pos":       branch.Pos,
			"condition": encodeExpr(branch.Cond),
			"body":      encodeStmtList(branch.Body),
		})
	}
	return out
}

func encodeExpr(expr Expr) any {
	switch e := expr.(type) {
	case *BinaryExpr:
		return map[string]any{"kind": "BinaryExpr", "pos": e.Pos, "op": tokenOp(e.Op), "left": encodeExpr(e.Left), "right": encodeExpr(e.Right)}
	case *UnaryExpr:
		return map[string]any{"kind": "UnaryExpr", "pos": e.Pos, "op": tokenOp(e.Op), "value": encodeExpr(e.Value)}
	case *LiteralExpr:
		return map[string]any{"kind": "LiteralExpr", "pos": e.Pos, "type": e.Type, "value": e.Value}
	case *VarExpr:
		return map[string]any{"kind": "VarExpr", "pos": e.Pos, "name": e.Name}
	case *StrCallExpr:
		return map[string]any{"kind": "ToStrCallExpr", "pos": e.Pos, "value": encodeExpr(e.Value)}
	case *InputCallExpr:
		return map[string]any{"kind": "InputCallExpr", "pos": e.Pos}
	case *IsIntCallExpr:
		return map[string]any{"kind": "IsIntCallExpr", "pos": e.Pos, "value": encodeExpr(e.Value)}
	case *ToIntCallExpr:
		return map[string]any{"kind": "ToIntCallExpr", "pos": e.Pos, "value": encodeExpr(e.Value)}
	case *IsDoubleCallExpr:
		return map[string]any{"kind": "IsDoubleCallExpr", "pos": e.Pos, "value": encodeExpr(e.Value)}
	case *ToDoubleCallExpr:
		return map[string]any{"kind": "ToDoubleCallExpr", "pos": e.Pos, "value": encodeExpr(e.Value)}
	default:
		return map[string]any{"kind": "UnknownExpr"}
	}
}

func decodeStmtList(raw []json.RawMessage) ([]Stmt, error) {
	stmts := make([]Stmt, 0, len(raw))
	for _, item := range raw {
		stmt, err := decodeStmt(item)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

func decodeStmt(raw json.RawMessage) (Stmt, error) {
	var kind astKind
	if err := json.Unmarshal(raw, &kind); err != nil {
		return nil, err
	}
	switch kind.Kind {
	case "VarDecl":
		var node struct {
			Pos   SourcePos       `json:"pos"`
			Name  string          `json:"name"`
			Type  Type            `json:"type"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		value, err := decodeExpr(node.Value)
		if err != nil {
			return nil, err
		}
		return &VarDecl{Pos: node.Pos, Name: node.Name, Type: node.Type, Value: value}, nil
	case "Assign":
		var node struct {
			Pos   SourcePos       `json:"pos"`
			Name  string          `json:"name"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		value, err := decodeExpr(node.Value)
		if err != nil {
			return nil, err
		}
		return &Assign{Pos: node.Pos, Name: node.Name, Value: value}, nil
	case "PrintStmt":
		var node struct {
			Pos     SourcePos       `json:"pos"`
			Newline bool            `json:"newline"`
			Value   json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		value, err := decodeExpr(node.Value)
		if err != nil {
			return nil, err
		}
		return &PrintStmt{Pos: node.Pos, Newline: node.Newline, Value: value}, nil
	case "ExitStmt":
		var node struct {
			Pos  SourcePos       `json:"pos"`
			Code json.RawMessage `json:"code"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		code, err := decodeExpr(node.Code)
		if err != nil {
			return nil, err
		}
		return &ExitStmt{Pos: node.Pos, Code: code}, nil
	case "BreakStmt":
		var node struct {
			Pos SourcePos `json:"pos"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		return &BreakStmt{Pos: node.Pos}, nil
	case "IfStmt":
		var node struct {
			Pos       SourcePos         `json:"pos"`
			Condition json.RawMessage   `json:"condition"`
			Then      []json.RawMessage `json:"then"`
			ElseIf    []json.RawMessage `json:"elseif"`
			Else      []json.RawMessage `json:"else"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		cond, err := decodeExpr(node.Condition)
		if err != nil {
			return nil, err
		}
		thenBody, err := decodeStmtList(node.Then)
		if err != nil {
			return nil, err
		}
		elseIf, err := decodeIfBranches(node.ElseIf)
		if err != nil {
			return nil, err
		}
		elseBody, err := decodeStmtList(node.Else)
		if err != nil {
			return nil, err
		}
		return &IfStmt{Pos: node.Pos, Cond: cond, Then: thenBody, ElseIf: elseIf, Else: elseBody}, nil
	case "WhileStmt":
		var node struct {
			Pos       SourcePos         `json:"pos"`
			Condition json.RawMessage   `json:"condition"`
			Body      []json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		cond, err := decodeExpr(node.Condition)
		if err != nil {
			return nil, err
		}
		body, err := decodeStmtList(node.Body)
		if err != nil {
			return nil, err
		}
		return &WhileStmt{Pos: node.Pos, Cond: cond, Body: body}, nil
	default:
		return nil, fmt.Errorf("unknown statement kind %q", kind.Kind)
	}
}

func decodeIfBranches(raw []json.RawMessage) ([]IfBranch, error) {
	branches := make([]IfBranch, 0, len(raw))
	for _, item := range raw {
		var node struct {
			Kind      string            `json:"kind"`
			Pos       SourcePos         `json:"pos"`
			Condition json.RawMessage   `json:"condition"`
			Body      []json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(item, &node); err != nil {
			return nil, err
		}
		if node.Kind != "IfBranch" {
			return nil, fmt.Errorf("unknown elseif branch kind %q", node.Kind)
		}
		cond, err := decodeExpr(node.Condition)
		if err != nil {
			return nil, err
		}
		body, err := decodeStmtList(node.Body)
		if err != nil {
			return nil, err
		}
		branches = append(branches, IfBranch{Pos: node.Pos, Cond: cond, Body: body})
	}
	return branches, nil
}

func decodeExpr(raw json.RawMessage) (Expr, error) {
	var kind astKind
	if err := json.Unmarshal(raw, &kind); err != nil {
		return nil, err
	}
	switch kind.Kind {
	case "BinaryExpr":
		var node struct {
			Pos   SourcePos       `json:"pos"`
			Op    string          `json:"op"`
			Left  json.RawMessage `json:"left"`
			Right json.RawMessage `json:"right"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		left, err := decodeExpr(node.Left)
		if err != nil {
			return nil, err
		}
		right, err := decodeExpr(node.Right)
		if err != nil {
			return nil, err
		}
		op, err := parseTokenOp(node.Op)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Pos: node.Pos, Op: op, Left: left, Right: right}, nil
	case "UnaryExpr":
		var node struct {
			Pos   SourcePos       `json:"pos"`
			Op    string          `json:"op"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		value, err := decodeExpr(node.Value)
		if err != nil {
			return nil, err
		}
		op, err := parseTokenOp(node.Op)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Pos: node.Pos, Op: op, Value: value}, nil
	case "LiteralExpr":
		var node struct {
			Pos   SourcePos `json:"pos"`
			Type  Type      `json:"type"`
			Value string    `json:"value"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		return &LiteralExpr{Pos: node.Pos, Type: node.Type, Value: node.Value}, nil
	case "VarExpr":
		var node struct {
			Pos  SourcePos `json:"pos"`
			Name string    `json:"name"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		return &VarExpr{Pos: node.Pos, Name: node.Name}, nil
	case "ToStrCallExpr":
		value, pos, err := decodeSingleValueExpr(raw)
		if err != nil {
			return nil, err
		}
		return &StrCallExpr{Pos: pos, Value: value}, nil
	case "InputCallExpr":
		var node struct {
			Pos SourcePos `json:"pos"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		return &InputCallExpr{Pos: node.Pos}, nil
	case "IsIntCallExpr":
		value, pos, err := decodeSingleValueExpr(raw)
		if err != nil {
			return nil, err
		}
		return &IsIntCallExpr{Pos: pos, Value: value}, nil
	case "ToIntCallExpr":
		value, pos, err := decodeSingleValueExpr(raw)
		if err != nil {
			return nil, err
		}
		return &ToIntCallExpr{Pos: pos, Value: value}, nil
	case "IsDoubleCallExpr":
		value, pos, err := decodeSingleValueExpr(raw)
		if err != nil {
			return nil, err
		}
		return &IsDoubleCallExpr{Pos: pos, Value: value}, nil
	case "ToDoubleCallExpr":
		value, pos, err := decodeSingleValueExpr(raw)
		if err != nil {
			return nil, err
		}
		return &ToDoubleCallExpr{Pos: pos, Value: value}, nil
	default:
		return nil, fmt.Errorf("unknown expression kind %q", kind.Kind)
	}
}

func decodeSingleValueExpr(raw json.RawMessage) (Expr, SourcePos, error) {
	var node struct {
		Pos   SourcePos       `json:"pos"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, SourcePos{}, err
	}
	value, err := decodeExpr(node.Value)
	if err != nil {
		return nil, SourcePos{}, err
	}
	return value, node.Pos, nil
}

func tokenOp(kind TokenKind) string {
	switch kind {
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenCaret:
		return "^"
	case TokenEqualEqual:
		return "=="
	case TokenBangEqual:
		return "!="
	case TokenLess:
		return "<"
	case TokenLessEqual:
		return "<="
	case TokenGreater:
		return ">"
	case TokenGreaterEqual:
		return ">="
	default:
		return "?"
	}
}

func parseTokenOp(op string) (TokenKind, error) {
	switch op {
	case "+":
		return TokenPlus, nil
	case "-":
		return TokenMinus, nil
	case "*":
		return TokenStar, nil
	case "/":
		return TokenSlash, nil
	case "%":
		return TokenPercent, nil
	case "^":
		return TokenCaret, nil
	case "==":
		return TokenEqualEqual, nil
	case "!=":
		return TokenBangEqual, nil
	case "<":
		return TokenLess, nil
	case "<=":
		return TokenLessEqual, nil
	case ">":
		return TokenGreater, nil
	case ">=":
		return TokenGreaterEqual, nil
	default:
		return TokenEOF, fmt.Errorf("unknown operator %q", op)
	}
}
