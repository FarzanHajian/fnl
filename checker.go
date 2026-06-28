// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"errors"
	"fmt"
)

type Checker struct {
	scopes    []map[string]Type
	loopDepth int
}

func NewChecker() *Checker {
	return &Checker{scopes: []map[string]Type{{}}}
}

func (c *Checker) Check(prog *Program) error {
	for _, stmt := range prog.Statements {
		if err := c.checkStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Checker) checkStmt(stmt Stmt) error {
	switch s := stmt.(type) {
	case *VarDecl:
		if _, exists := c.current()[s.Name]; exists {
			return errorAt(s.Pos, "variable %q already declared in this scope", s.Name)
		}
		valueType, err := c.exprType(s.Value)
		if err != nil {
			return err
		}
		if !canAssign(s.Type, valueType) {
			return errorAt(s.Pos, "cannot assign %s expression to %s variable %q", valueType, s.Type, s.Name)
		}
		c.current()[s.Name] = s.Type
	case *Assign:
		targetType, ok := c.lookup(s.Name)
		if !ok {
			return errorAt(s.Pos, "assignment to undeclared variable %q", s.Name)
		}
		valueType, err := c.exprType(s.Value)
		if err != nil {
			return err
		}
		if !canAssign(targetType, valueType) {
			return errorAt(s.Pos, "cannot assign %s expression to %s variable %q", valueType, targetType, s.Name)
		}
	case *PrintStmt:
		typ, err := c.exprType(s.Value)
		if err != nil {
			return err
		}
		if typ != TypeString {
			return errorAt(s.Pos, "print expects string, got %s", typ)
		}
	case *ExitStmt:
		typ, err := c.exprType(s.Code)
		if err != nil {
			return err
		}
		if typ != TypeInt {
			return errorAt(s.Pos, "exit expects int, got %s", typ)
		}
	case *BreakStmt:
		if c.loopDepth == 0 {
			return errorAt(s.Pos, "break is only allowed inside while")
		}
	case *IfStmt:
		if err := c.checkCondition(s.Cond, "if"); err != nil {
			return err
		}
		if err := c.checkBlock(s.Then); err != nil {
			return err
		}
		for _, branch := range s.ElseIf {
			if err := c.checkCondition(branch.Cond, "elseif"); err != nil {
				return err
			}
			if err := c.checkBlock(branch.Body); err != nil {
				return err
			}
		}
		if len(s.Else) > 0 {
			if err := c.checkBlock(s.Else); err != nil {
				return err
			}
		}
	case *WhileStmt:
		if err := c.checkCondition(s.Cond, "while"); err != nil {
			return err
		}
		c.loopDepth++
		defer func() {
			c.loopDepth--
		}()
		return c.checkBlock(s.Body)
	}
	return nil
}

func (c *Checker) checkCondition(expr Expr, name string) error {
	typ, err := c.exprType(expr)
	if err != nil {
		return err
	}
	if typ != TypeBool {
		return errorAt(exprPos(expr), "%s condition must be bool, got %s", name, typ)
	}
	return nil
}

func (c *Checker) checkBlock(stmts []Stmt) error {
	c.push()
	defer c.pop()
	for _, stmt := range stmts {
		if err := c.checkStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Checker) exprType(expr Expr) (Type, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		return e.Type, nil
	case *VarExpr:
		typ, ok := c.lookup(e.Name)
		if !ok {
			return "", errorAt(e.Pos, "use of undeclared variable %q", e.Name)
		}
		return typ, nil
	case *StrCallExpr:
		if _, err := c.exprType(e.Value); err != nil {
			return "", err
		}
		return TypeString, nil
	case *InputCallExpr:
		return TypeString, nil
	case *IsIntCallExpr:
		typ, err := c.exprType(e.Value)
		if err != nil {
			return "", err
		}
		if typ != TypeString {
			return "", errorAt(e.Pos, "is_int expects string, got %s", typ)
		}
		return TypeBool, nil
	case *ToIntCallExpr:
		typ, err := c.exprType(e.Value)
		if err != nil {
			return "", err
		}
		if typ != TypeString {
			return "", errorAt(e.Pos, "to_int expects string, got %s", typ)
		}
		return TypeInt, nil
	case *IsDoubleCallExpr:
		typ, err := c.exprType(e.Value)
		if err != nil {
			return "", err
		}
		if typ != TypeString {
			return "", errorAt(e.Pos, "is_double expects string, got %s", typ)
		}
		return TypeBool, nil
	case *ToDoubleCallExpr:
		typ, err := c.exprType(e.Value)
		if err != nil {
			return "", err
		}
		if typ != TypeString {
			return "", errorAt(e.Pos, "to_double expects string, got %s", typ)
		}
		return TypeDouble, nil
	case *UnaryExpr:
		typ, err := c.exprType(e.Value)
		if err != nil {
			return "", err
		}
		if typ != TypeInt && typ != TypeDouble {
			return "", errorAt(e.Pos, "unary '-' expects numeric expression, got %s", typ)
		}
		return typ, nil
	case *BinaryExpr:
		return c.binaryType(e)
	default:
		return "", errors.New("unknown expression")
	}
}

func (c *Checker) binaryType(e *BinaryExpr) (Type, error) {
	left, err := c.exprType(e.Left)
	if err != nil {
		return "", err
	}
	right, err := c.exprType(e.Right)
	if err != nil {
		return "", err
	}
	switch e.Op {
	case TokenPlus:
		if left == TypeString || right == TypeString {
			if left == TypeString && right == TypeString {
				return TypeString, nil
			}
			return "", errorAt(e.Pos, "string concatenation requires string + string, got %s + %s", left, right)
		}
		return c.numericBinaryType(e.Pos, left, right, "+")
	case TokenMinus, TokenStar, TokenSlash:
		return c.numericBinaryType(e.Pos, left, right, opName(e.Op))
	case TokenPercent:
		if left == TypeInt && right == TypeInt {
			return TypeInt, nil
		}
		return "", errorAt(e.Pos, "%% requires int %% int, got %s %% %s", left, right)
	case TokenCaret:
		return c.numericBinaryType(e.Pos, left, right, "^")
	case TokenEqualEqual, TokenBangEqual, TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual:
		if left != right {
			return "", errorAt(e.Pos, "comparison requires matching types, got %s and %s", left, right)
		}
		if left == TypeString {
			return TypeBool, nil
		}
		if e.Op != TokenEqualEqual && e.Op != TokenBangEqual && left == TypeBool {
			return "", errorAt(e.Pos, "ordering comparison is not supported for bool")
		}
		return TypeBool, nil
	default:
		return "", errors.New("unknown binary operator")
	}
}

func (c *Checker) numericBinaryType(pos SourcePos, left, right Type, op string) (Type, error) {
	if !isNumeric(left) || !isNumeric(right) {
		return "", errorAt(pos, "%s requires numeric operands, got %s and %s", op, left, right)
	}
	if left == TypeDouble || right == TypeDouble {
		return TypeDouble, nil
	}
	return TypeInt, nil
}

func errorAt(pos SourcePos, format string, args ...any) error {
	return fmt.Errorf("line %d:%d: %s", pos.Line, pos.Col, fmt.Sprintf(format, args...))
}

func exprPos(expr Expr) SourcePos {
	switch e := expr.(type) {
	case *LiteralExpr:
		return e.Pos
	case *VarExpr:
		return e.Pos
	case *StrCallExpr:
		return e.Pos
	case *InputCallExpr:
		return e.Pos
	case *IsIntCallExpr:
		return e.Pos
	case *ToIntCallExpr:
		return e.Pos
	case *IsDoubleCallExpr:
		return e.Pos
	case *ToDoubleCallExpr:
		return e.Pos
	case *UnaryExpr:
		return e.Pos
	case *BinaryExpr:
		return e.Pos
	default:
		return SourcePos{}
	}
}

func canAssign(target, value Type) bool {
	return target == value || target == TypeDouble && value == TypeInt
}

func isNumeric(typ Type) bool {
	return typ == TypeInt || typ == TypeDouble
}

func opName(kind TokenKind) string {
	switch kind {
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	default:
		return "operator"
	}
}

func (c *Checker) push() {
	c.scopes = append(c.scopes, map[string]Type{})
}

func (c *Checker) pop() {
	c.scopes = c.scopes[:len(c.scopes)-1]
}

func (c *Checker) current() map[string]Type {
	return c.scopes[len(c.scopes)-1]
}

func (c *Checker) lookup(name string) (Type, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if typ, ok := c.scopes[i][name]; ok {
			return typ, true
		}
	}
	return "", false
}
