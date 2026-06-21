// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strconv"
	"strings"
)

type CBinding struct {
	typ Type
}

type CExpr struct {
	typ  Type
	code string
}

type CCodegen struct {
	scopes []map[string]CBinding
	out    strings.Builder
}

func NewCCodegen() *CCodegen {
	return &CCodegen{scopes: []map[string]CBinding{{}}}
}

func GenerateC(prog *Program) (string, error) {
	return NewCCodegen().Generate(prog)
}

func (g *CCodegen) Generate(prog *Program) (string, error) {
	g.out.WriteString(cRuntime())
	g.line("int main(void) {")
	g.pushScope()
	for _, stmt := range prog.Statements {
		if err := g.stmt(stmt); err != nil {
			return "", err
		}
	}
	g.line("return 0;")
	g.popScope()
	g.line("}")
	return g.out.String(), nil
}

func (g *CCodegen) stmt(stmt Stmt) error {
	switch s := stmt.(type) {
	case *VarDecl:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		value = g.convert(value, s.Type)
		g.current()[s.Name] = CBinding{typ: s.Type}
		g.line("%s %s = %s;", cType(s.Type), s.Name, value.code)
	case *Assign:
		binding, ok := g.lookup(s.Name)
		if !ok {
			return fmt.Errorf("internal error: unknown variable %q during C generation", s.Name)
		}
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		value = g.convert(value, binding.typ)
		g.line("%s = %s;", s.Name, value.code)
	case *PrintStmt:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		if s.Newline {
			g.line("printf(\"%%s\\n\", %s);", value.code)
		} else {
			g.line("printf(\"%%s\", %s);", value.code)
		}
	case *ExitStmt:
		code, err := g.expr(s.Code)
		if err != nil {
			return err
		}
		g.line("exit((int)(%s));", code.code)
	case *BreakStmt:
		g.line("break;")
	case *IfStmt:
		cond, err := g.expr(s.Cond)
		if err != nil {
			return err
		}
		g.line("if (%s) {", cond.code)
		g.pushScope()
		for _, inner := range s.Then {
			if err := g.stmt(inner); err != nil {
				return err
			}
		}
		g.popScope()
		for _, branch := range s.ElseIf {
			cond, err := g.expr(branch.Cond)
			if err != nil {
				return err
			}
			g.line("} else if (%s) {", cond.code)
			g.pushScope()
			for _, inner := range branch.Body {
				if err := g.stmt(inner); err != nil {
					return err
				}
			}
			g.popScope()
		}
		if len(s.Else) > 0 {
			g.line("} else {")
			g.pushScope()
			for _, inner := range s.Else {
				if err := g.stmt(inner); err != nil {
					return err
				}
			}
			g.popScope()
		}
		g.line("}")
	case *WhileStmt:
		cond, err := g.expr(s.Cond)
		if err != nil {
			return err
		}
		g.line("while (%s) {", cond.code)
		g.pushScope()
		for _, inner := range s.Body {
			if err := g.stmt(inner); err != nil {
				return err
			}
		}
		g.popScope()
		g.line("}")
	}
	return nil
}

func (g *CCodegen) expr(expr Expr) (CExpr, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		switch e.Type {
		case TypeString:
			return CExpr{typ: TypeString, code: "fnl_strdup(" + quoteCString(e.Value) + ")"}, nil
		case TypeBool:
			if e.Value == "true" {
				return CExpr{typ: TypeBool, code: "1"}, nil
			}
			return CExpr{typ: TypeBool, code: "0"}, nil
		case TypeDouble:
			return CExpr{typ: TypeDouble, code: cDoubleLiteral(e.Value)}, nil
		default:
			return CExpr{typ: TypeInt64, code: e.Value}, nil
		}
	case *VarExpr:
		binding, ok := g.lookup(e.Name)
		if !ok {
			return CExpr{}, fmt.Errorf("internal error: unknown variable %q during C generation", e.Name)
		}
		return CExpr{typ: binding.typ, code: e.Name}, nil
	case *StrCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: TypeString, code: cStrCall(value)}, nil
	case *InputCallExpr:
		return CExpr{typ: TypeString, code: "fnl_input()"}, nil
	case *IsInt64CallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: TypeBool, code: "fnl_is_int64(" + value.code + ")"}, nil
	case *ToInt64CallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: TypeInt64, code: "fnl_to_int64(" + value.code + ")"}, nil
	case *IsDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: TypeBool, code: "fnl_is_double(" + value.code + ")"}, nil
	case *ToDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: TypeDouble, code: "fnl_to_double(" + value.code + ")"}, nil
	case *UnaryExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		return CExpr{typ: value.typ, code: "(-" + value.code + ")"}, nil
	case *BinaryExpr:
		return g.binaryExpr(e)
	default:
		return CExpr{}, fmt.Errorf("unknown expression")
	}
}

func (g *CCodegen) binaryExpr(e *BinaryExpr) (CExpr, error) {
	left, err := g.expr(e.Left)
	if err != nil {
		return CExpr{}, err
	}
	right, err := g.expr(e.Right)
	if err != nil {
		return CExpr{}, err
	}
	switch e.Op {
	case TokenPlus:
		if left.typ == TypeString {
			return CExpr{typ: TypeString, code: fmt.Sprintf("fnl_str_concat(%s, %s)", left.code, right.code)}, nil
		}
		resultType := numericResult(left.typ, right.typ)
		left = g.convert(left, resultType)
		right = g.convert(right, resultType)
		return CExpr{typ: resultType, code: fmt.Sprintf("(%s + %s)", left.code, right.code)}, nil
	case TokenMinus, TokenStar, TokenSlash:
		resultType := numericResult(left.typ, right.typ)
		left = g.convert(left, resultType)
		right = g.convert(right, resultType)
		return CExpr{typ: resultType, code: fmt.Sprintf("(%s %s %s)", left.code, cOp(e.Op), right.code)}, nil
	case TokenPercent:
		return CExpr{typ: TypeInt64, code: fmt.Sprintf("(%s %% %s)", left.code, right.code)}, nil
	case TokenCaret:
		resultType := numericResult(left.typ, right.typ)
		if resultType == TypeInt64 {
			return CExpr{typ: TypeInt64, code: fmt.Sprintf("fnl_pow_i64(%s, %s)", left.code, right.code)}, nil
		}
		left = g.convert(left, TypeDouble)
		right = g.convert(right, TypeDouble)
		return CExpr{typ: TypeDouble, code: fmt.Sprintf("pow(%s, %s)", left.code, right.code)}, nil
	case TokenEqualEqual, TokenBangEqual, TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual:
		if left.typ == TypeString {
			op := "=="
			if e.Op == TokenBangEqual {
				op = "!="
			}
			return CExpr{typ: TypeBool, code: fmt.Sprintf("(strcmp(%s, %s) %s 0)", left.code, right.code, op)}, nil
		}
		return CExpr{typ: TypeBool, code: fmt.Sprintf("(%s %s %s)", left.code, cOp(e.Op), right.code)}, nil
	default:
		return CExpr{}, fmt.Errorf("unknown binary operator")
	}
}

func (g *CCodegen) convert(value CExpr, target Type) CExpr {
	if value.typ == target {
		return value
	}
	if value.typ == TypeInt64 && target == TypeDouble {
		return CExpr{typ: TypeDouble, code: "(double)(" + value.code + ")"}
	}
	return value
}

func cStrCall(value CExpr) string {
	switch value.typ {
	case TypeString:
		return "fnl_strdup(" + value.code + ")"
	case TypeInt64:
		return "fnl_str_i64(" + value.code + ")"
	case TypeDouble:
		return "fnl_str_double(" + value.code + ")"
	case TypeBool:
		return "fnl_str_bool(" + value.code + ")"
	default:
		return value.code
	}
}

func numericResult(left, right Type) Type {
	if left == TypeDouble || right == TypeDouble {
		return TypeDouble
	}
	return TypeInt64
}

func cType(typ Type) string {
	switch typ {
	case TypeDouble:
		return "double"
	case TypeBool:
		return "int"
	case TypeString:
		return "char*"
	default:
		return "int64_t"
	}
}

func cOp(kind TokenKind) string {
	switch kind {
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
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

func cDoubleLiteral(value string) string {
	if strings.ContainsAny(value, ".eE") {
		return value
	}
	return value + ".0"
}

func quoteCString(s string) string {
	return strconv.Quote(s)
}

func (g *CCodegen) line(format string, args ...any) {
	g.out.WriteString(strings.Repeat("    ", len(g.scopes)-1))
	g.out.WriteString(fmt.Sprintf(format, args...))
	g.out.WriteByte('\n')
}

func (g *CCodegen) pushScope() {
	g.scopes = append(g.scopes, map[string]CBinding{})
}

func (g *CCodegen) popScope() {
	g.scopes = g.scopes[:len(g.scopes)-1]
}

func (g *CCodegen) current() map[string]CBinding {
	return g.scopes[len(g.scopes)-1]
}

func (g *CCodegen) lookup(name string) (CBinding, bool) {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if binding, ok := g.scopes[i][name]; ok {
			return binding, true
		}
	}
	return CBinding{}, false
}

func cRuntime() string {
	return `#include <ctype.h>
#include <errno.h>
#include <math.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char* fnl_strdup(const char* s) {
    size_t n = strlen(s) + 1;
    char* out = (char*)malloc(n);
    if (!out) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }
    memcpy(out, s, n);
    return out;
}

static char* fnl_format(size_t n, const char* fmt, ...) {
    va_list args;
    va_start(args, fmt);
    char* out = (char*)malloc(n);
    if (!out) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }
    vsnprintf(out, n, fmt, args);
    va_end(args);
    return out;
}

static char* fnl_str_i64(int64_t value) {
    return fnl_format(64, "%lld", (long long)value);
}

static char* fnl_str_double(double value) {
    return fnl_format(128, "%.15g", value);
}

static char* fnl_str_bool(int value) {
    return fnl_strdup(value ? "true" : "false");
}

static char* fnl_str_concat(const char* a, const char* b) {
    size_t an = strlen(a);
    size_t bn = strlen(b);
    char* out = (char*)malloc(an + bn + 1);
    if (!out) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }
    memcpy(out, a, an);
    memcpy(out + an, b, bn + 1);
    return out;
}

static char* fnl_input(void) {
    size_t capacity = 64;
    size_t length = 0;
    char* out = (char*)malloc(capacity);
    if (!out) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }

    int ch;
    while ((ch = getchar()) != EOF && ch != '\n') {
        if (length + 1 >= capacity) {
            capacity *= 2;
            char* next = (char*)realloc(out, capacity);
            if (!next) {
                free(out);
                fprintf(stderr, "out of memory\n");
                exit(1);
            }
            out = next;
        }
        out[length++] = (char)ch;
    }
    out[length] = '\0';
    return out;
}

static int fnl_is_int64(const char* s) {
    if (!s || *s == '\0' || isspace((unsigned char)*s)) {
        return 0;
    }

    errno = 0;
    char* end = NULL;
    (void)strtoll(s, &end, 10);
    return end != s && *end == '\0' && errno != ERANGE;
}

static int64_t fnl_to_int64(const char* s) {
    if (!fnl_is_int64(s)) {
        return 0;
    }
    return (int64_t)strtoll(s, NULL, 10);
}

static int64_t fnl_pow_i64(int64_t base, int64_t exponent) {
    if (exponent < 0) {
        return 0;
    }

    int64_t result = 1;
    while (exponent > 0) {
        if (exponent % 2 == 1) {
            result *= base;
        }
        exponent /= 2;
        if (exponent > 0) {
            base *= base;
        }
    }
    return result;
}

static int fnl_is_double(const char* s) {
    if (!s || *s == '\0' || isspace((unsigned char)*s)) {
        return 0;
    }

    errno = 0;
    char* end = NULL;
    (void)strtod(s, &end);
    return end != s && *end == '\0' && errno != ERANGE;
}

static double fnl_to_double(const char* s) {
    if (!fnl_is_double(s)) {
        return 0.0;
    }
    return strtod(s, NULL);
}

`
}
