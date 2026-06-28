// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strconv"
	"strings"
)

type LLVMBinding struct {
	typ Type
	ptr string
}

type LLVMValue struct {
	typ  Type
	name string
}

type LLVMCodegen struct {
	scopes       []map[string]LLVMBinding
	out          strings.Builder
	globals      strings.Builder
	tempID       int
	labelID      int
	loopEndStack []string
	terminated   bool
}

func NewLLVMCodegen() *LLVMCodegen {
	return &LLVMCodegen{scopes: []map[string]LLVMBinding{{}}}
}

func GenerateLLVM(prog *Program) (string, error) {
	return NewLLVMCodegen().Generate(prog)
}

func (g *LLVMCodegen) Generate(prog *Program) (string, error) {
	g.line("define i32 @main() {")
	g.label("entry")
	for _, stmt := range prog.Statements {
		if g.terminated {
			break
		}
		if err := g.stmt(stmt); err != nil {
			return "", err
		}
	}
	if !g.terminated {
		g.line("  ret i32 0")
	}
	g.line("}")

	var module strings.Builder
	module.WriteString(fmt.Sprintf("target triple = %q\n\n", hostLLVMTriple()))
	module.WriteString("@fmt_print = private unnamed_addr constant [3 x i8] c\"%s\\00\"\n")
	module.WriteString("@fmt_println = private unnamed_addr constant [4 x i8] c\"%s\\0A\\00\"\n")
	module.WriteString(g.globals.String())
	module.WriteString("\n")
	module.WriteString("declare i32 @printf(ptr, ...)\n")
	module.WriteString("declare void @exit(i32)\n")
	module.WriteString("declare double @pow(double, double)\n")
	module.WriteString("declare i32 @strcmp(ptr, ptr)\n")
	module.WriteString(llvmRuntime())
	module.WriteString(g.out.String())
	return module.String(), nil
}

func (g *LLVMCodegen) stmt(stmt Stmt) error {
	switch s := stmt.(type) {
	case *VarDecl:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		ptr := g.temp()
		g.line("  %s = alloca %s", ptr, llvmType(s.Type))
		g.line("  store %s %s, ptr %s", llvmType(s.Type), value.name, ptr)
		g.current()[s.Name] = LLVMBinding{typ: s.Type, ptr: ptr}
	case *Assign:
		binding, ok := g.lookup(s.Name)
		if !ok {
			return fmt.Errorf("internal error: unknown variable %q during LLVM generation", s.Name)
		}
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.line("  store %s %s, ptr %s", llvmType(binding.typ), value.name, binding.ptr)
	case *PrintStmt:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.line("  %s = call i32 (ptr, ...) @printf(ptr %s, ptr %s)", g.temp(), g.printFormatPtr(s.Newline), value.name)
	case *ExitStmt:
		code, err := g.expr(s.Code)
		if err != nil {
			return err
		}
		exitCode := code.name
		if code.typ == TypeInt {
			if n, err := strconv.ParseInt(code.name, 10, 32); err == nil {
				exitCode = strconv.FormatInt(n, 10)
			} else {
				exitCode = g.temp()
				g.line("  %s = trunc i64 %s to i32", exitCode, code.name)
			}
		}
		g.line("  call void @exit(i32 %s)", exitCode)
		g.line("  unreachable")
		g.terminated = true
	case *BreakStmt:
		if len(g.loopEndStack) == 0 {
			return fmt.Errorf("internal error: break outside loop during LLVM generation")
		}
		g.line("  br label %%%s", g.loopEndStack[len(g.loopEndStack)-1])
		g.terminated = true
	case *IfStmt:
		return g.ifStmt(s)
	case *WhileStmt:
		condLabel := g.newLabel("while.cond")
		bodyLabel := g.newLabel("while.body")
		endLabel := g.newLabel("while.end")
		g.line("  br label %%%s", condLabel)
		g.label(condLabel)
		cond, err := g.expr(s.Cond)
		if err != nil {
			return err
		}
		g.line("  br i1 %s, label %%%s, label %%%s", cond.name, bodyLabel, endLabel)
		g.label(bodyLabel)
		g.loopEndStack = append(g.loopEndStack, endLabel)
		g.pushScope()
		for _, inner := range s.Body {
			if g.terminated {
				break
			}
			if err := g.stmt(inner); err != nil {
				return err
			}
		}
		g.popScope()
		g.loopEndStack = g.loopEndStack[:len(g.loopEndStack)-1]
		if !g.terminated {
			g.line("  br label %%%s", condLabel)
		}
		g.label(endLabel)
		g.terminated = false
	}
	return nil
}

func (g *LLVMCodegen) expr(expr Expr) (LLVMValue, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		switch e.Type {
		case TypeString:
			name, size := g.stringConstant(e.Value)
			ptr := g.temp()
			g.line("  %s = call ptr @fnl_strdup(ptr getelementptr inbounds ([%d x i8], ptr %s, i64 0, i64 0))", ptr, size, name)
			return LLVMValue{typ: TypeString, name: ptr}, nil
		case TypeBool:
			if e.Value == "true" {
				return LLVMValue{typ: TypeBool, name: "true"}, nil
			}
			return LLVMValue{typ: TypeBool, name: "false"}, nil
		case TypeDouble:
			return LLVMValue{typ: TypeDouble, name: llvmDoubleLiteral(e.Value)}, nil
		default:
			return LLVMValue{typ: TypeInt, name: e.Value}, nil
		}
	case *VarExpr:
		binding, ok := g.lookup(e.Name)
		if !ok {
			return LLVMValue{}, fmt.Errorf("internal error: unknown variable %q during LLVM generation", e.Name)
		}
		value := g.temp()
		g.line("  %s = load %s, ptr %s", value, llvmType(binding.typ), binding.ptr)
		return LLVMValue{typ: binding.typ, name: value}, nil
	case *StrCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		return g.strCall(value), nil
	case *InputCallExpr:
		result := g.temp()
		g.line("  %s = call ptr @fnl_input()", result)
		return LLVMValue{typ: TypeString, name: result}, nil
	case *IsIntCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		result := g.temp()
		if value.typ == TypeDouble {
			g.line("  %s = call i1 @fnl_is_int_double(double %s)", result, value.name)
			return LLVMValue{typ: TypeBool, name: result}, nil
		}
		g.line("  %s = call i1 @fnl_is_int(ptr %s)", result, value.name)
		return LLVMValue{typ: TypeBool, name: result}, nil
	case *ToIntCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		result := g.temp()
		if value.typ == TypeDouble {
			g.line("  %s = fptosi double %s to i64", result, value.name)
			return LLVMValue{typ: TypeInt, name: result}, nil
		}
		g.line("  %s = call i64 @fnl_to_int(ptr %s)", result, value.name)
		return LLVMValue{typ: TypeInt, name: result}, nil
	case *IsDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		result := g.temp()
		if value.typ == TypeInt {
			g.line("  %s = call i1 @fnl_is_double_int(i64 %s)", result, value.name)
			return LLVMValue{typ: TypeBool, name: result}, nil
		}
		g.line("  %s = call i1 @fnl_is_double(ptr %s)", result, value.name)
		return LLVMValue{typ: TypeBool, name: result}, nil
	case *ToDoubleCallExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		if value.typ == TypeInt {
			return g.convert(value, TypeDouble)
		}
		result := g.temp()
		g.line("  %s = call double @fnl_to_double(ptr %s)", result, value.name)
		return LLVMValue{typ: TypeDouble, name: result}, nil
	case *UnaryExpr:
		value, err := g.expr(e.Value)
		if err != nil {
			return LLVMValue{}, err
		}
		result := g.temp()
		if value.typ == TypeDouble {
			g.line("  %s = fsub double 0.000000e+00, %s", result, value.name)
		} else {
			g.line("  %s = sub i64 0, %s", result, value.name)
		}
		return LLVMValue{typ: value.typ, name: result}, nil
	case *BinaryExpr:
		return g.binaryExpr(e)
	default:
		return LLVMValue{}, fmt.Errorf("unknown expression")
	}
}

func (g *LLVMCodegen) ifStmt(stmt *IfStmt) error {
	endLabel := g.newLabel("if.end")
	nextLabel := g.newLabel("if.next")
	cond, err := g.expr(stmt.Cond)
	if err != nil {
		return err
	}
	thenLabel := g.newLabel("if.then")
	g.line("  br i1 %s, label %%%s, label %%%s", cond.name, thenLabel, nextLabel)
	g.label(thenLabel)
	thenTerminated, err := g.scopedStmtList(stmt.Then)
	if err != nil {
		return err
	}
	if !thenTerminated {
		g.line("  br label %%%s", endLabel)
	}

	allBranchesTerminate := thenTerminated
	g.label(nextLabel)
	for i, branch := range stmt.ElseIf {
		cond, err := g.expr(branch.Cond)
		if err != nil {
			return err
		}
		thenLabel := g.newLabel("elseif.then")
		nextLabel = endLabel
		if i < len(stmt.ElseIf)-1 || len(stmt.Else) > 0 {
			nextLabel = g.newLabel("elseif.next")
		}
		g.line("  br i1 %s, label %%%s, label %%%s", cond.name, thenLabel, nextLabel)
		g.label(thenLabel)
		branchTerminated, err := g.scopedStmtList(branch.Body)
		if err != nil {
			return err
		}
		if !branchTerminated {
			g.line("  br label %%%s", endLabel)
		}
		allBranchesTerminate = allBranchesTerminate && branchTerminated
		if nextLabel != endLabel {
			g.label(nextLabel)
		}
	}

	if len(stmt.Else) > 0 {
		elseTerminated, err := g.scopedStmtList(stmt.Else)
		if err != nil {
			return err
		}
		if !elseTerminated {
			g.line("  br label %%%s", endLabel)
		}
		allBranchesTerminate = allBranchesTerminate && elseTerminated
	} else {
		if len(stmt.ElseIf) == 0 {
			g.line("  br label %%%s", endLabel)
		}
		allBranchesTerminate = false
	}

	g.label(endLabel)
	if allBranchesTerminate {
		g.line("  unreachable")
	}
	g.terminated = allBranchesTerminate
	return nil
}

func (g *LLVMCodegen) scopedStmtList(stmts []Stmt) (bool, error) {
	g.pushScope()
	for _, inner := range stmts {
		if g.terminated {
			break
		}
		if err := g.stmt(inner); err != nil {
			g.popScope()
			return false, err
		}
	}
	terminated := g.terminated
	g.popScope()
	return terminated, nil
}

func (g *LLVMCodegen) binaryExpr(e *BinaryExpr) (LLVMValue, error) {
	left, err := g.expr(e.Left)
	if err != nil {
		return LLVMValue{}, err
	}
	right, err := g.expr(e.Right)
	if err != nil {
		return LLVMValue{}, err
	}
	switch e.Op {
	case TokenPlus:
		if left.typ == TypeString {
			result := g.temp()
			g.line("  %s = call ptr @fnl_str_concat(ptr %s, ptr %s)", result, left.name, right.name)
			return LLVMValue{typ: TypeString, name: result}, nil
		}
		return g.numericBinary(left, right, e.Op)
	case TokenMinus, TokenStar, TokenSlash:
		return g.numericBinary(left, right, e.Op)
	case TokenPercent:
		result := g.temp()
		g.line("  %s = srem i64 %s, %s", result, left.name, right.name)
		return LLVMValue{typ: TypeInt, name: result}, nil
	case TokenCaret:
		resultType := numericResult(left.typ, right.typ)
		result := g.temp()
		if resultType == TypeInt {
			g.line("  %s = call i64 @fnl_pow_int(i64 %s, i64 %s)", result, left.name, right.name)
			return LLVMValue{typ: TypeInt, name: result}, nil
		}
		left, _ = g.convert(left, TypeDouble)
		right, _ = g.convert(right, TypeDouble)
		g.line("  %s = call double @pow(double %s, double %s)", result, left.name, right.name)
		return LLVMValue{typ: TypeDouble, name: result}, nil
	case TokenEqualEqual, TokenBangEqual, TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual:
		if left.typ == TypeString {
			cmpResult := g.temp()
			boolResult := g.temp()
			g.line("  %s = call i32 @strcmp(ptr %s, ptr %s)", cmpResult, left.name, right.name)
			g.line("  %s = icmp %s i32 %s, 0", boolResult, llvmIntCmp(e.Op), cmpResult)
			return LLVMValue{typ: TypeBool, name: boolResult}, nil
		}
		if isNumeric(left.typ) && isNumeric(right.typ) {
			resultType := numericResult(left.typ, right.typ)
			left, err = g.convert(left, resultType)
			if err != nil {
				return LLVMValue{}, err
			}
			right, err = g.convert(right, resultType)
			if err != nil {
				return LLVMValue{}, err
			}
		}
		result := g.temp()
		if left.typ == TypeDouble {
			g.line("  %s = fcmp %s double %s, %s", result, llvmFloatCmp(e.Op), left.name, right.name)
		} else {
			g.line("  %s = icmp %s %s %s, %s", result, llvmIntCmp(e.Op), llvmType(left.typ), left.name, right.name)
		}
		return LLVMValue{typ: TypeBool, name: result}, nil
	default:
		return LLVMValue{}, fmt.Errorf("unknown binary operator")
	}
}

func (g *LLVMCodegen) numericBinary(left, right LLVMValue, op TokenKind) (LLVMValue, error) {
	resultType := numericResult(left.typ, right.typ)
	left, err := g.convert(left, resultType)
	if err != nil {
		return LLVMValue{}, err
	}
	right, err = g.convert(right, resultType)
	if err != nil {
		return LLVMValue{}, err
	}
	result := g.temp()
	g.line("  %s = %s %s %s, %s", result, llvmBinaryOp(op, resultType), llvmType(resultType), left.name, right.name)
	return LLVMValue{typ: resultType, name: result}, nil
}

func (g *LLVMCodegen) strCall(value LLVMValue) LLVMValue {
	if value.typ == TypeString {
		result := g.temp()
		g.line("  %s = call ptr @fnl_strdup(ptr %s)", result, value.name)
		return LLVMValue{typ: TypeString, name: result}
	}
	result := g.temp()
	switch value.typ {
	case TypeInt:
		g.line("  %s = call ptr @fnl_str_int(i64 %s)", result, value.name)
	case TypeDouble:
		g.line("  %s = call ptr @fnl_str_double(double %s)", result, value.name)
	case TypeBool:
		g.line("  %s = call ptr @fnl_str_bool(i1 %s)", result, value.name)
	}
	return LLVMValue{typ: TypeString, name: result}
}

func (g *LLVMCodegen) convert(value LLVMValue, target Type) (LLVMValue, error) {
	if value.typ == target {
		return value, nil
	}
	if value.typ == TypeInt && target == TypeDouble {
		result := g.temp()
		g.line("  %s = sitofp i64 %s to double", result, value.name)
		return LLVMValue{typ: TypeDouble, name: result}, nil
	}
	return LLVMValue{}, fmt.Errorf("cannot convert %s to %s", value.typ, target)
}

func (g *LLVMCodegen) stringConstant(value string) (string, int) {
	escaped := llvmStringBytes(value)
	size := len([]byte(value)) + 1
	name := fmt.Sprintf("@str.%d", g.tempID)
	g.globals.WriteString(fmt.Sprintf("%s = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n", name, size, escaped))
	return name, size
}

func (g *LLVMCodegen) printFormatPtr(newline bool) string {
	if newline {
		return "getelementptr inbounds ([4 x i8], ptr @fmt_println, i64 0, i64 0)"
	}
	return "getelementptr inbounds ([3 x i8], ptr @fmt_print, i64 0, i64 0)"
}

func (g *LLVMCodegen) temp() string {
	name := fmt.Sprintf("%%t%d", g.tempID)
	g.tempID++
	return name
}

func (g *LLVMCodegen) newLabel(prefix string) string {
	label := fmt.Sprintf("%s.%d", prefix, g.labelID)
	g.labelID++
	return label
}

func (g *LLVMCodegen) label(name string) {
	g.out.WriteString(name)
	g.out.WriteString(":\n")
	g.terminated = false
}

func (g *LLVMCodegen) line(format string, args ...any) {
	g.out.WriteString(fmt.Sprintf(format, args...))
	g.out.WriteByte('\n')
}

func (g *LLVMCodegen) pushScope() {
	g.scopes = append(g.scopes, map[string]LLVMBinding{})
}

func (g *LLVMCodegen) popScope() {
	g.scopes = g.scopes[:len(g.scopes)-1]
}

func (g *LLVMCodegen) current() map[string]LLVMBinding {
	return g.scopes[len(g.scopes)-1]
}

func (g *LLVMCodegen) lookup(name string) (LLVMBinding, bool) {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if binding, ok := g.scopes[i][name]; ok {
			return binding, true
		}
	}
	return LLVMBinding{}, false
}

func llvmType(typ Type) string {
	switch typ {
	case TypeDouble:
		return "double"
	case TypeBool:
		return "i1"
	case TypeString:
		return "ptr"
	default:
		return "i64"
	}
}

func llvmDoubleLiteral(value string) string {
	if strings.ContainsAny(value, ".eE") {
		return value
	}
	return value + ".0"
}

func llvmBinaryOp(op TokenKind, typ Type) string {
	if typ == TypeDouble {
		switch op {
		case TokenPlus:
			return "fadd"
		case TokenMinus:
			return "fsub"
		case TokenStar:
			return "fmul"
		case TokenSlash:
			return "fdiv"
		}
	}
	switch op {
	case TokenPlus:
		return "add"
	case TokenMinus:
		return "sub"
	case TokenStar:
		return "mul"
	case TokenSlash:
		return "sdiv"
	default:
		return "add"
	}
}

func llvmIntCmp(op TokenKind) string {
	switch op {
	case TokenEqualEqual:
		return "eq"
	case TokenBangEqual:
		return "ne"
	case TokenLess:
		return "slt"
	case TokenLessEqual:
		return "sle"
	case TokenGreater:
		return "sgt"
	case TokenGreaterEqual:
		return "sge"
	default:
		return "eq"
	}
}

func llvmFloatCmp(op TokenKind) string {
	switch op {
	case TokenEqualEqual:
		return "oeq"
	case TokenBangEqual:
		return "one"
	case TokenLess:
		return "olt"
	case TokenLessEqual:
		return "ole"
	case TokenGreater:
		return "ogt"
	case TokenGreaterEqual:
		return "oge"
	default:
		return "oeq"
	}
}

func llvmStringBytes(value string) string {
	var b strings.Builder
	for _, by := range []byte(value) {
		switch by {
		case '\\':
			b.WriteString(`\5C`)
		case '"':
			b.WriteString(`\22`)
		case '\n':
			b.WriteString(`\0A`)
		case '\t':
			b.WriteString(`\09`)
		default:
			if by < 32 || by >= 127 {
				b.WriteString(`\`)
				b.WriteString(strings.ToUpper(strconv.FormatInt(int64(by), 16)))
			} else {
				b.WriteByte(by)
			}
		}
	}
	return b.String()
}

func llvmRuntime() string {
	return `
@fnl_fmt_int = private unnamed_addr constant [5 x i8] c"%lld\00"
@fnl_fmt_double = private unnamed_addr constant [6 x i8] c"%.15g\00"
@fnl_true = private unnamed_addr constant [5 x i8] c"true\00"
@fnl_false = private unnamed_addr constant [6 x i8] c"false\00"

declare ptr @malloc(i64)
declare ptr @realloc(ptr, i64)
declare void @free(ptr)
declare i64 @strlen(ptr)
declare ptr @memcpy(ptr, ptr, i64)
declare i32 @snprintf(ptr, i64, ptr, ...)
declare i32 @getchar()
declare i32 @isspace(i32)
declare i64 @strtoll(ptr, ptr, i32)
declare double @strtod(ptr, ptr)

define ptr @fnl_strdup(ptr %s) {
entry:
  %len = call i64 @strlen(ptr %s)
  %n = add i64 %len, 1
  %out = call ptr @malloc(i64 %n)
  %is_null = icmp eq ptr %out, null
  br i1 %is_null, label %oom, label %copy

oom:
  call void @exit(i32 1)
  unreachable

copy:
  %ignored = call ptr @memcpy(ptr %out, ptr %s, i64 %n)
  ret ptr %out
}

define ptr @fnl_str_int(i64 %value) {
entry:
  %out = call ptr @malloc(i64 64)
  %is_null = icmp eq ptr %out, null
  br i1 %is_null, label %oom, label %format

oom:
  call void @exit(i32 1)
  unreachable

format:
  %ignored = call i32 (ptr, i64, ptr, ...) @snprintf(ptr %out, i64 64, ptr getelementptr inbounds ([5 x i8], ptr @fnl_fmt_int, i64 0, i64 0), i64 %value)
  ret ptr %out
}

define ptr @fnl_str_double(double %value) {
entry:
  %out = call ptr @malloc(i64 128)
  %is_null = icmp eq ptr %out, null
  br i1 %is_null, label %oom, label %format

oom:
  call void @exit(i32 1)
  unreachable

format:
  %ignored = call i32 (ptr, i64, ptr, ...) @snprintf(ptr %out, i64 128, ptr getelementptr inbounds ([6 x i8], ptr @fnl_fmt_double, i64 0, i64 0), double %value)
  ret ptr %out
}

define ptr @fnl_str_bool(i1 %value) {
entry:
  br i1 %value, label %true_value, label %false_value

true_value:
  %true_copy = call ptr @fnl_strdup(ptr getelementptr inbounds ([5 x i8], ptr @fnl_true, i64 0, i64 0))
  ret ptr %true_copy

false_value:
  %false_copy = call ptr @fnl_strdup(ptr getelementptr inbounds ([6 x i8], ptr @fnl_false, i64 0, i64 0))
  ret ptr %false_copy
}

define ptr @fnl_str_concat(ptr %a, ptr %b) {
entry:
  %an = call i64 @strlen(ptr %a)
  %bn = call i64 @strlen(ptr %b)
  %sum = add i64 %an, %bn
  %n = add i64 %sum, 1
  %out = call ptr @malloc(i64 %n)
  %is_null = icmp eq ptr %out, null
  br i1 %is_null, label %oom, label %copy_a

oom:
  call void @exit(i32 1)
  unreachable

copy_a:
  %ignored_a = call ptr @memcpy(ptr %out, ptr %a, i64 %an)
  %tail = getelementptr i8, ptr %out, i64 %an
  %bn_with_null = add i64 %bn, 1
  %ignored_b = call ptr @memcpy(ptr %tail, ptr %b, i64 %bn_with_null)
  ret ptr %out
}

define ptr @fnl_input() {
entry:
  %capacity = alloca i64
  %length = alloca i64
  %out_slot = alloca ptr
  store i64 64, ptr %capacity
  store i64 0, ptr %length
  %out = call ptr @malloc(i64 64)
  %is_null = icmp eq ptr %out, null
  br i1 %is_null, label %oom, label %init

oom:
  call void @exit(i32 1)
  unreachable

init:
  store ptr %out, ptr %out_slot
  br label %loop

loop:
  %ch = call i32 @getchar()
  %is_eof = icmp eq i32 %ch, -1
  %is_newline = icmp eq i32 %ch, 10
  %done = or i1 %is_eof, %is_newline
  br i1 %done, label %finish, label %ensure_capacity

ensure_capacity:
  %len = load i64, ptr %length
  %cap = load i64, ptr %capacity
  %needed = add i64 %len, 1
  %full = icmp uge i64 %needed, %cap
  br i1 %full, label %resize, label %store_char

resize:
  %old_cap = load i64, ptr %capacity
  %new_cap = mul i64 %old_cap, 2
  %old_out = load ptr, ptr %out_slot
  %next = call ptr @realloc(ptr %old_out, i64 %new_cap)
  %next_is_null = icmp eq ptr %next, null
  br i1 %next_is_null, label %resize_oom, label %resize_ok

resize_oom:
  %to_free = load ptr, ptr %out_slot
  call void @free(ptr %to_free)
  call void @exit(i32 1)
  unreachable

resize_ok:
  store ptr %next, ptr %out_slot
  store i64 %new_cap, ptr %capacity
  br label %store_char

store_char:
  %cur_out = load ptr, ptr %out_slot
  %cur_len = load i64, ptr %length
  %dst = getelementptr i8, ptr %cur_out, i64 %cur_len
  %byte = trunc i32 %ch to i8
  store i8 %byte, ptr %dst
  %new_len = add i64 %cur_len, 1
  store i64 %new_len, ptr %length
  br label %loop

finish:
  %final_out = load ptr, ptr %out_slot
  %final_len = load i64, ptr %length
  %terminator = getelementptr i8, ptr %final_out, i64 %final_len
  store i8 0, ptr %terminator
  ret ptr %final_out
}

define i1 @fnl_is_int(ptr %s) {
entry:
  %is_null = icmp eq ptr %s, null
  br i1 %is_null, label %false, label %check_first

check_first:
  %first = load i8, ptr %s
  %is_empty = icmp eq i8 %first, 0
  br i1 %is_empty, label %false, label %check_space

check_space:
  %first_i32 = zext i8 %first to i32
  %space = call i32 @isspace(i32 %first_i32)
  %is_space = icmp ne i32 %space, 0
  br i1 %is_space, label %false, label %parse

parse:
  %end_slot = alloca ptr
  %ignored = call i64 @strtoll(ptr %s, ptr %end_slot, i32 10)
  %end = load ptr, ptr %end_slot
  %no_digits = icmp eq ptr %end, %s
  br i1 %no_digits, label %false, label %check_end

check_end:
  %end_ch = load i8, ptr %end
  %at_end = icmp eq i8 %end_ch, 0
  br i1 %at_end, label %true, label %false

true:
  ret i1 true

false:
  ret i1 false
}

define i64 @fnl_to_int(ptr %s) {
entry:
  %ok = call i1 @fnl_is_int(ptr %s)
  br i1 %ok, label %parse, label %zero

parse:
  %value = call i64 @strtoll(ptr %s, ptr null, i32 10)
  ret i64 %value

zero:
  ret i64 0
}

define i1 @fnl_is_int_double(double %value) {
entry:
  %ordered = fcmp ord double %value, 0.000000e+00
  br i1 %ordered, label %check_min, label %false

check_min:
  %ge_min = fcmp oge double %value, -9.223372036854776e+18
  br i1 %ge_min, label %check_max, label %false

check_max:
  %lt_max = fcmp olt double %value, 9.223372036854776e+18
  br i1 %lt_max, label %true, label %false

true:
  ret i1 true

false:
  ret i1 false
}

define i1 @fnl_is_double_int(i64 %value) {
entry:
  %as_double = sitofp i64 %value to double
  %too_high = fcmp oge double %as_double, 9.223372036854776e+18
  br i1 %too_high, label %false, label %round_trip

round_trip:
  %back = fptosi double %as_double to i64
  %ok = icmp eq i64 %back, %value
  ret i1 %ok

false:
  ret i1 false
}

define i1 @fnl_is_double(ptr %s) {
entry:
  %is_null = icmp eq ptr %s, null
  br i1 %is_null, label %false, label %check_first

check_first:
  %first = load i8, ptr %s
  %is_empty = icmp eq i8 %first, 0
  br i1 %is_empty, label %false, label %check_space

check_space:
  %first_i32 = zext i8 %first to i32
  %space = call i32 @isspace(i32 %first_i32)
  %is_space = icmp ne i32 %space, 0
  br i1 %is_space, label %false, label %parse

parse:
  %end_slot = alloca ptr
  %ignored = call double @strtod(ptr %s, ptr %end_slot)
  %end = load ptr, ptr %end_slot
  %no_digits = icmp eq ptr %end, %s
  br i1 %no_digits, label %false, label %check_end

check_end:
  %end_ch = load i8, ptr %end
  %at_end = icmp eq i8 %end_ch, 0
  br i1 %at_end, label %true, label %false

true:
  ret i1 true

false:
  ret i1 false
}

define double @fnl_to_double(ptr %s) {
entry:
  %ok = call i1 @fnl_is_double(ptr %s)
  br i1 %ok, label %parse, label %zero

parse:
  %value = call double @strtod(ptr %s, ptr null)
  ret double %value

zero:
  ret double 0.000000e+00
}

define i64 @fnl_pow_int(i64 %base_arg, i64 %exponent_arg) {
entry:
  %is_negative = icmp slt i64 %exponent_arg, 0
  br i1 %is_negative, label %return_zero, label %init

return_zero:
  ret i64 0

init:
  %result_slot = alloca i64
  %base_slot = alloca i64
  %exponent_slot = alloca i64
  store i64 1, ptr %result_slot
  store i64 %base_arg, ptr %base_slot
  store i64 %exponent_arg, ptr %exponent_slot
  br label %loop

loop:
  %exponent = load i64, ptr %exponent_slot
  %keep_going = icmp sgt i64 %exponent, 0
  br i1 %keep_going, label %body, label %done

body:
  %remainder = srem i64 %exponent, 2
  %is_odd = icmp eq i64 %remainder, 1
  br i1 %is_odd, label %multiply_result, label %after_result

multiply_result:
  %current_result = load i64, ptr %result_slot
  %current_base = load i64, ptr %base_slot
  %next_result = mul i64 %current_result, %current_base
  store i64 %next_result, ptr %result_slot
  br label %after_result

after_result:
  %current_exponent = load i64, ptr %exponent_slot
  %half_exponent = sdiv i64 %current_exponent, 2
  store i64 %half_exponent, ptr %exponent_slot
  %need_square = icmp sgt i64 %half_exponent, 0
  br i1 %need_square, label %square_base, label %loop

square_base:
  %base = load i64, ptr %base_slot
  %squared = mul i64 %base, %base
  store i64 %squared, ptr %base_slot
  br label %loop

done:
  %result = load i64, ptr %result_slot
  ret i64 %result
}

`
}
