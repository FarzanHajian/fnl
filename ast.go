// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

type Type string

const (
	TypeInt    Type = "int"
	TypeDouble Type = "double"
	TypeBool   Type = "bool"
	TypeString Type = "string"
)

type Program struct {
	Statements []Stmt
}

type SourcePos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type Stmt interface {
	stmt()
}

type VarDecl struct {
	Pos   SourcePos
	Name  string
	Type  Type
	Value Expr
}

type Assign struct {
	Pos   SourcePos
	Name  string
	Value Expr
}

type PrintStmt struct {
	Pos     SourcePos
	Value   Expr
	Newline bool
}

type ExitStmt struct {
	Pos  SourcePos
	Code Expr
}

type BreakStmt struct {
	Pos SourcePos
}

type IfBranch struct {
	Pos  SourcePos
	Cond Expr
	Body []Stmt
}

type IfStmt struct {
	Pos    SourcePos
	Cond   Expr
	Then   []Stmt
	ElseIf []IfBranch
	Else   []Stmt
}

type WhileStmt struct {
	Pos  SourcePos
	Cond Expr
	Body []Stmt
}

func (*VarDecl) stmt()   {}
func (*Assign) stmt()    {}
func (*PrintStmt) stmt() {}
func (*ExitStmt) stmt()  {}
func (*BreakStmt) stmt() {}
func (*IfStmt) stmt()    {}
func (*WhileStmt) stmt() {}

type Expr interface {
	expr()
}

type BinaryExpr struct {
	Pos   SourcePos
	Left  Expr
	Op    TokenKind
	Right Expr
}

type UnaryExpr struct {
	Pos   SourcePos
	Op    TokenKind
	Value Expr
}

type LiteralExpr struct {
	Pos   SourcePos
	Value string
	Type  Type
}

type VarExpr struct {
	Pos  SourcePos
	Name string
}

type StrCallExpr struct {
	Pos   SourcePos
	Value Expr
}

type InputCallExpr struct {
	Pos SourcePos
}

type IsIntCallExpr struct {
	Pos   SourcePos
	Value Expr
}

type ToIntCallExpr struct {
	Pos   SourcePos
	Value Expr
}

type IsDoubleCallExpr struct {
	Pos   SourcePos
	Value Expr
}

type ToDoubleCallExpr struct {
	Pos   SourcePos
	Value Expr
}

type MathRandomCallExpr struct {
	Pos SourcePos
}

func (*BinaryExpr) expr()         {}
func (*UnaryExpr) expr()          {}
func (*LiteralExpr) expr()        {}
func (*VarExpr) expr()            {}
func (*StrCallExpr) expr()        {}
func (*InputCallExpr) expr()      {}
func (*IsIntCallExpr) expr()      {}
func (*ToIntCallExpr) expr()      {}
func (*IsDoubleCallExpr) expr()   {}
func (*ToDoubleCallExpr) expr()   {}
func (*MathRandomCallExpr) expr() {}
