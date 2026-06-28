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
		g.current()[s.Name] = CBinding{typ: s.Type}
		g.line("%s %s = %s;", cType(s.Type), s.Name, value.code)
	case *Assign:
		if _, ok := g.lookup(s.Name); !ok {
			return fmt.Errorf("internal error: unknown variable %q during C generation", s.Name)
		}
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.line("%s = %s;", s.Name, value.code)
	case *PrintStmt:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		if s.Newline {
			g.line("fnl_println(%s);", value.code)
		} else {
			g.line("fnl_print(%s);", value.code)
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
			return CExpr{typ: TypeInt, code: e.Value}, nil
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
	case *IsIntCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		if value.typ == TypeDouble {
			return CExpr{typ: TypeBool, code: "fnl_is_int_double(" + value.code + ")"}, nil
		}
		return CExpr{typ: TypeBool, code: "fnl_is_int(" + value.code + ")"}, nil
	case *ToIntCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		if value.typ == TypeDouble {
			return CExpr{typ: TypeInt, code: "(int64_t)(" + value.code + ")"}, nil
		}
		return CExpr{typ: TypeInt, code: "fnl_to_int(" + value.code + ")"}, nil
	case *IsDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		if value.typ == TypeInt {
			return CExpr{typ: TypeBool, code: "fnl_is_double_int(" + value.code + ")"}, nil
		}
		return CExpr{typ: TypeBool, code: "fnl_is_double(" + value.code + ")"}, nil
	case *ToDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return CExpr{}, err
		}
		if value.typ == TypeInt {
			return g.convert(value, TypeDouble), nil
		}
		return CExpr{typ: TypeDouble, code: "fnl_to_double(" + value.code + ")"}, nil
	case *MathRandomCallExpr:
		return CExpr{typ: TypeDouble, code: "fnl_math_random()"}, nil
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
		return CExpr{typ: TypeInt, code: fmt.Sprintf("(%s %% %s)", left.code, right.code)}, nil
	case TokenCaret:
		resultType := numericResult(left.typ, right.typ)
		if resultType == TypeInt {
			return CExpr{typ: TypeInt, code: fmt.Sprintf("fnl_pow_int(%s, %s)", left.code, right.code)}, nil
		}
		left = g.convert(left, TypeDouble)
		right = g.convert(right, TypeDouble)
		return CExpr{typ: TypeDouble, code: fmt.Sprintf("pow(%s, %s)", left.code, right.code)}, nil
	case TokenEqualEqual, TokenBangEqual, TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual:
		if left.typ == TypeString {
			return CExpr{typ: TypeBool, code: fmt.Sprintf("(fnl_str_cmp(%s, %s) %s 0)", left.code, right.code, cOp(e.Op))}, nil
		}
		if isNumeric(left.typ) && isNumeric(right.typ) {
			resultType := numericResult(left.typ, right.typ)
			left = g.convert(left, resultType)
			right = g.convert(right, resultType)
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
	if value.typ == TypeInt && target == TypeDouble {
		return CExpr{typ: TypeDouble, code: "(double)(" + value.code + ")"}
	}
	return value
}

func cStrCall(value CExpr) string {
	switch value.typ {
	case TypeString:
		return "fnl_str_copy(" + value.code + ")"
	case TypeInt:
		return "fnl_str_int(" + value.code + ")"
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
	return TypeInt
}

func cType(typ Type) string {
	switch typ {
	case TypeDouble:
		return "double"
	case TypeBool:
		return "int"
	case TypeString:
		return "fnl_string"
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
#include <time.h>
#ifdef _WIN32
#include <io.h>
#include <windows.h>
#endif

typedef struct {
    char* data;
    int64_t len;
} fnl_string;

static fnl_string fnl_string_from_bytes(const char* data, size_t len) {
    fnl_string out;
    out.data = (char*)malloc(len + 1);
    if (!out.data) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }
    memcpy(out.data, data, len);
    out.data[len] = '\0';
    out.len = (int64_t)len;
    return out;
}

static fnl_string fnl_strdup(const char* s) {
    return fnl_string_from_bytes(s, strlen(s));
}

static fnl_string fnl_str_copy(fnl_string s) {
    return fnl_string_from_bytes(s.data, (size_t)s.len);
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

static fnl_string fnl_str_from_owned_cstr(char* s) {
    fnl_string out;
    out.data = s;
    out.len = (int64_t)strlen(s);
    return out;
}

static fnl_string fnl_str_int(int64_t value) {
    return fnl_str_from_owned_cstr(fnl_format(64, "%lld", (long long)value));
}

static fnl_string fnl_str_double(double value) {
    return fnl_str_from_owned_cstr(fnl_format(128, "%.15g", value));
}

static fnl_string fnl_str_bool(int value) {
    return fnl_strdup(value ? "true" : "false");
}

static fnl_string fnl_str_concat(fnl_string a, fnl_string b) {
    fnl_string out;
    out.len = a.len + b.len;
    out.data = (char*)malloc((size_t)out.len + 1);
    if (!out.data) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }
    memcpy(out.data, a.data, (size_t)a.len);
    memcpy(out.data + a.len, b.data, (size_t)b.len);
    out.data[out.len] = '\0';
    return out;
}

static int fnl_str_cmp(fnl_string a, fnl_string b) {
    size_t min_len = a.len < b.len ? (size_t)a.len : (size_t)b.len;
    int cmp = memcmp(a.data, b.data, min_len);
    if (cmp != 0) {
        return cmp;
    }
    if (a.len < b.len) {
        return -1;
    }
    if (a.len > b.len) {
        return 1;
    }
    return 0;
}

static void fnl_print(fnl_string s) {
#ifdef _WIN32
    if (_isatty(_fileno(stdout))) {
        HANDLE out = GetStdHandle(STD_OUTPUT_HANDLE);
        if (out != INVALID_HANDLE_VALUE) {
            int needed = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, s.data, (int)s.len, NULL, 0);
            if (needed > 0) {
                wchar_t* wide = (wchar_t*)malloc((size_t)needed * sizeof(wchar_t));
                if (!wide) {
                    fprintf(stderr, "out of memory\n");
                    exit(1);
                }
                MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, s.data, (int)s.len, wide, needed);
                DWORD written = 0;
                WriteConsoleW(out, wide, (DWORD)needed, &written, NULL);
                free(wide);
                return;
            }
        }
    }
#endif
    fwrite(s.data, 1, (size_t)s.len, stdout);
}

static void fnl_println(fnl_string s) {
    fnl_print(s);
    fputc('\n', stdout);
}

static fnl_string fnl_input(void) {
    size_t capacity = 64;
    size_t length = 0;
    fnl_string out;
    out.data = (char*)malloc(capacity);
    if (!out.data) {
        fprintf(stderr, "out of memory\n");
        exit(1);
    }

    int ch;
    while ((ch = getchar()) != EOF && ch != '\n') {
        if (length + 1 >= capacity) {
            capacity *= 2;
            char* next = (char*)realloc(out.data, capacity);
            if (!next) {
                free(out.data);
                fprintf(stderr, "out of memory\n");
                exit(1);
            }
            out.data = next;
        }
        out.data[length++] = (char)ch;
    }
    out.data[length] = '\0';
    out.len = (int64_t)length;
    return out;
}

static int fnl_is_int(fnl_string s) {
    if (!s.data || s.len == 0 || isspace((unsigned char)s.data[0])) {
        return 0;
    }

    errno = 0;
    char* end = NULL;
    (void)strtoll(s.data, &end, 10);
    return end != s.data && *end == '\0' && errno != ERANGE;
}

static int64_t fnl_to_int(fnl_string s) {
    if (!fnl_is_int(s)) {
        return 0;
    }
    return (int64_t)strtoll(s.data, NULL, 10);
}

static int fnl_is_int_double(double value) {
    return isfinite(value) && value >= -9223372036854775808.0 && value < 9223372036854775808.0;
}

static int fnl_is_double_int(int64_t value) {
    return (long double)(double)value == (long double)value;
}

static int64_t fnl_pow_int(int64_t base, int64_t exponent) {
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

static int fnl_is_double(fnl_string s) {
    if (!s.data || s.len == 0 || isspace((unsigned char)s.data[0])) {
        return 0;
    }

    errno = 0;
    char* end = NULL;
    (void)strtod(s.data, &end);
    return end != s.data && *end == '\0' && errno != ERANGE;
}

static double fnl_to_double(fnl_string s) {
    if (!fnl_is_double(s)) {
        return 0.0;
    }
    return strtod(s.data, NULL);
}

static unsigned fnl_random_seed_from_time(void) {
#ifdef _WIN32
    ULONGLONG ms = GetTickCount64();
    return (unsigned)(ms ^ (ms >> 32));
#else
    struct timespec ts;
    if (timespec_get(&ts, TIME_UTC) == TIME_UTC) {
        uint64_t ms = (uint64_t)ts.tv_sec * 1000ULL + (uint64_t)(ts.tv_nsec / 1000000L);
        return (unsigned)(ms ^ (ms >> 32));
    }
    return (unsigned)time(NULL);
#endif
}

static double fnl_math_random(void) {
    static int seeded = 0;
    if (!seeded) {
        srand(fnl_random_seed_from_time());
        seeded = 1;
    }
    return (double)rand() / ((double)RAND_MAX + 1.0);
}

`
}
