// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndGenerateCForLanguageFeatures(t *testing.T) {
	src := strings.Join([]string{
		`var name:string="FNL"`,
		`var x:int=2`,
		`var y:int=5`,
		`var ok:bool=x<y`,
		`print("hello " + name)`,
		`if ok {`,
		`print(to_str(x^y))`,
		`} else {`,
		`print("no")`,
		`}`,
		`while x<y {`,
		`x=x+1`,
		`}`,
	}, "\n")
	prog, err := ParseAndCheckSource(src)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_str_concat", "fnl_pow_int", "while", "fnl_print(", "WriteConsoleW"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
}

func TestGenerateLLVMForLanguageFeatures(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var s:string="a"`,
		`var x:int=1`,
		`var y:int=2`,
		`print(s + to_str(x))`,
		`if x<y {`,
		`print("yes")`,
		`}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"target triple", "@fnl_str_concat", "icmp slt", "br i1"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestASTJSONRoundTrip(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var x:int=1`,
		`if x==1 {`,
		`println("one")`,
		`} elseif x==2 {`,
		`println("two")`,
		`} else {`,
		`println("other")`,
		`}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseAndCheckSource returned error: %v", err)
	}
	data, err := ExportAST(prog)
	if err != nil {
		t.Fatalf("ExportAST returned error: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("exported AST is not valid JSON: %v", err)
	}
	if root["format"] != "fnl.ast" || root["kind"] != "Program" {
		t.Fatalf("unexpected AST root: %v", root)
	}
	if !strings.Contains(string(data), `"kind": "IfStmt"`) || !strings.Contains(string(data), `"kind": "IfBranch"`) {
		t.Fatalf("exported AST missing expected kind fields:\n%s", data)
	}
	imported, err := ImportAST(data)
	if err != nil {
		t.Fatalf("ImportAST returned error: %v", err)
	}
	if err := NewChecker().Check(imported); err != nil {
		t.Fatalf("imported AST failed semantic checking: %v", err)
	}
	csrc, err := GenerateC(imported)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	if !strings.Contains(csrc, "} else if ((x == 2)) {") {
		t.Fatalf("generated C from imported AST missing elseif:\n%s", csrc)
	}
}

func TestASTGraphExport(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var x:int=1`,
		`if x==1 {`,
		`println("one")`,
		`} elseif x==2 {`,
		`println("two")`,
		`}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseAndCheckSource returned error: %v", err)
	}
	dot := string(ExportASTGraph(prog))
	for _, want := range []string{
		"digraph FNL_AST",
		`label="Program"`,
		`label="IfStmt"`,
		`label="IfBranch\nelseif 1"`,
		`label="condition"`,
	} {
		if !strings.Contains(dot, want) {
			t.Fatalf("DOT output missing %q:\n%s", want, dot)
		}
	}
}

func TestTypeRules(t *testing.T) {
	tests := []string{
		`print(1)`,
		`if 1 {` + "\n" + `print("x")` + "\n" + `}`,
		`var x:int=true`,
		`var x:double=1`,
		`var x:int=1%2.0`,
		`var x:string="a"+1`,
		`var x:bool=true<false`,
		`var x:int64=1`,
		`var x:int=to_int64("1")`,
	}
	for _, src := range tests {
		if _, err := ParseAndCheckSource(src); err == nil {
			t.Fatalf("expected type error for source:\n%s", src)
		}
	}
}

func TestPowerUsesNumericPromotion(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var int_power:int=2^3`,
		`var double_power:double=2.0^3`,
		`var promoted_power:double=2^3.0`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"int64_t int_power = fnl_pow_int(2, 3);", "double double_power = pow(2.0, (double)(3));", "double promoted_power = pow((double)(2), 3.0);"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define i64 @fnl_pow_int(i64 %base_arg, i64 %exponent_arg)", "call i64 @fnl_pow_int", "call double @pow"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestStrictAssignmentButMixedNumericExpressionsPromote(t *testing.T) {
	_, err := ParseAndCheckSource(strings.Join([]string{
		`var i:int=10`,
		`var result:double=10+i`,
	}, "\n"))
	if err == nil {
		t.Fatal("expected int expression assigned to double to be rejected")
	}

	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var d:double=10.0`,
		`var result:double=10+d`,
		`var ok:bool=10<d`,
	}, "\n"))
	if err != nil {
		t.Fatalf("mixed numeric expression should promote: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"double result = ((double)(10) + d);", "int ok = ((double)(10) < d);"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
}

func TestSemanticErrorsIncludeLineAndColumn(t *testing.T) {
	_, err := ParseAndCheckSource(strings.Join([]string{
		`var num:int=0`,
		`num=1.5`,
	}, "\n"))
	if err == nil {
		t.Fatal("expected type error")
	}
	if !strings.Contains(err.Error(), `line 2:1: cannot assign double expression to int variable "num"`) {
		t.Fatalf("semantic error should include line and column, got: %v", err)
	}
}

func TestStringComparisons(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var a:string="same"`,
		`var b:string="same"`,
		`var c:string="other"`,
		`var d:string="sane"`,
		`println(to_str(a==b))`,
		`println(to_str(a!=c))`,
		`println(to_str(d<a))`,
		`println(to_str(a>=b))`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_str_cmp(a, b) == 0", "fnl_str_cmp(a, c) != 0", "fnl_str_cmp(d, a) < 0", "fnl_str_cmp(a, b) >= 0"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"declare i32 @strcmp(ptr, ptr)", "call i32 @strcmp", "icmp eq i32", "icmp ne i32", "icmp slt i32", "icmp sge i32"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestIntParsingBuiltins(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var good:string="123"`,
		`var bad:string="12x"`,
		`var ok:bool=is_int(good)`,
		`var not_ok:bool=is_int(bad)`,
		`var value:int=to_int(good)`,
		`println(to_str(ok))`,
		`println(to_str(not_ok))`,
		`println(to_str(value))`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_is_int(good)", "fnl_is_int(bad)", "fnl_to_int(good)"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define i1 @fnl_is_int(ptr %s)", "define i64 @fnl_to_int(ptr %s)", "call i1 @fnl_is_int", "call i64 @fnl_to_int"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestDoubleParsingBuiltins(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var good:string="12.5"`,
		`var bad:string="12x"`,
		`var ok:bool=is_double(good)`,
		`var not_ok:bool=is_double(bad)`,
		`var value:double=to_double(good)`,
		`var exact:bool=is_double(42)`,
		`var from_int:double=to_double(42)`,
		`println(to_str(ok))`,
		`println(to_str(not_ok))`,
		`println(to_str(value))`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_is_double(good)", "fnl_is_double(bad)", "fnl_to_double(good)", "fnl_is_double_int(42)", "(double)(42)"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define i1 @fnl_is_double(ptr %s)", "define i1 @fnl_is_double_int(i64 %value)", "define double @fnl_to_double(ptr %s)", "call i1 @fnl_is_double", "call i1 @fnl_is_double_int", "call double @fnl_to_double", "sitofp i64 42 to double"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestMathRandomBuiltin(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var r:double=math_random()`,
		`var secret:int=1+to_int(math_random()*to_double(10))`,
		`println(to_str(r))`,
		`println(to_str(secret))`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_math_random()", "srand(fnl_random_seed_from_time())"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define double @fnl_math_random()", "call double @fnl_math_random()", "call void @srand"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
	data, err := ExportAST(prog)
	if err != nil {
		t.Fatalf("ExportAST returned error: %v", err)
	}
	if !strings.Contains(string(data), `"kind": "MathRandomCallExpr"`) {
		t.Fatalf("exported AST missing MathRandomCallExpr:\n%s", data)
	}
	imported, err := ImportAST(data)
	if err != nil {
		t.Fatalf("ImportAST returned error: %v", err)
	}
	if err := NewChecker().Check(imported); err != nil {
		t.Fatalf("imported AST failed semantic checking: %v", err)
	}
}

func TestNumericConversionOverloads(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var d:double=12.75`,
		`var can_fit:bool=is_int(d)`,
		`var i:int=to_int(d)`,
		`println(to_str(can_fit))`,
		`println(to_str(i))`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"fnl_is_int_double(d)", "(int64_t)(d)"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define i1 @fnl_is_int_double(double %value)", "call i1 @fnl_is_int_double", "fptosi double"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestConversionBuiltinsRejectUnsupportedTypes(t *testing.T) {
	tests := []string{
		`var ok:bool=is_int(true)`,
		`var value:int=to_int(true)`,
		`var ok:bool=is_double(true)`,
		`var value:double=to_double(true)`,
	}
	for _, src := range tests {
		if _, err := ParseAndCheckSource(src); err == nil {
			t.Fatalf("expected type error for source:\n%s", src)
		}
	}
}

func TestStrAliasIsRejected(t *testing.T) {
	_, err := ParseAndCheckSource(`println(str(1))`)
	if err == nil {
		t.Fatal("expected str alias to be rejected")
	}
}

func TestInputReturnsString(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var name:string=input()`,
		`println("hello " + name)`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	if !strings.Contains(csrc, "fnl_string name = fnl_input();") {
		t.Fatalf("generated C missing input call:\n%s", csrc)
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"define ptr @fnl_input()", "call ptr @fnl_input()"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestBreakAndExitStatements(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var x:int=0`,
		`while x<10 {`,
		`x=x+1`,
		`if x==3 {`,
		`break`,
		`}`,
		`}`,
		`exit(0)`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"break;", "exit((int)(0));"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"declare void @exit(i32)", "br label %while.end", "call void @exit(i32 0)", "unreachable"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestBreakRequiresLoopAndExitRequiresInt(t *testing.T) {
	tests := []string{
		`break`,
		`exit("no")`,
		`exit(1.5)`,
	}
	for _, src := range tests {
		if _, err := ParseAndCheckSource(src); err == nil {
			t.Fatalf("expected semantic error for source:\n%s", src)
		}
	}
}

func TestParseArgsDefaultsAndFlags(t *testing.T) {
	opts, err := ParseArgs([]string{"examples/basics.fnl", "-o", filepath.Join("outputs", "app.exe"), "--emit-c", "--emit-llvm", "--emit-ast", "--emit-ast-graph", "--backend=clang"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.Output != filepath.Join("outputs", "app.exe") || !opts.EmitC || !opts.EmitLLVM || !opts.EmitAST || !opts.EmitGraph || opts.Backend != "clang" {
		t.Fatalf("unexpected options: %+v", opts)
	}
}

func TestParseArgsAcceptsASTInput(t *testing.T) {
	opts, err := ParseArgs([]string{filepath.Join("outputs", "basics.fnl.ast")})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.Output != "basics"+hostExeExt() {
		t.Fatalf("unexpected default output for AST input: %+v", opts)
	}
}

func TestParseArgsRejectsTargetOption(t *testing.T) {
	_, err := ParseArgs([]string{"examples/basics.fnl", "--target=linux-x64"})
	if err == nil {
		t.Fatal("expected --target to be rejected")
	}
	if !strings.Contains(err.Error(), "unknown option --target=linux-x64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompilerArgsDoNotForceClangTarget(t *testing.T) {
	args := compilerArgs("clang", "input.c", "app.exe")
	for _, arg := range args {
		if strings.HasPrefix(arg, "--target=") {
			t.Fatalf("compilerArgs should not force a target, got %v", args)
		}
	}
}

func TestClangArgsOnWindowsDoNotLinkUnixMathLibrary(t *testing.T) {
	args := compilerArgs("clang", "input.c", "app.exe")
	if hostExeExt() != ".exe" {
		t.Skip("Windows-specific clang/MSVC behavior")
	}
	for _, arg := range args {
		if arg == "-lm" {
			t.Fatalf("clang on Windows should not link Unix libm, got %v", args)
		}
	}
}

func TestMSVCToolchainCheck(t *testing.T) {
	getenv := func(key string) string {
		if key == "VCToolsInstallDir" {
			return `C:\BuildTools\VC\Tools\MSVC\14.0`
		}
		return ""
	}
	lookPath := func(string) (string, error) {
		return "", filepath.ErrBadPattern
	}
	if !hasMSVCToolchain(getenv, lookPath) {
		t.Fatal("expected VCToolsInstallDir to satisfy MSVC toolchain check")
	}
}

func TestIfWithoutElseDoesNotConsumeFollowingStatementNewline(t *testing.T) {
	_, err := ParseAndCheckSource(strings.Join([]string{
		`var x:int=0`,
		`if x==0 {`,
		`print("zero")`,
		`}`,
		`while x<2 {`,
		`x=x+1`,
		`}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
}

func TestElseIfBranches(t *testing.T) {
	prog, err := ParseAndCheckSource(strings.Join([]string{
		`var x:int=2`,
		`if x==1 {`,
		`println("one")`,
		`} elseif x==2 {`,
		`println("two")`,
		`} elseif x==3 {`,
		`println("three")`,
		`} else {`,
		`println("other")`,
		`}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{"} else if ((x == 2)) {", "} else if ((x == 3)) {", "} else {"} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
	ll, err := GenerateLLVM(prog)
	if err != nil {
		t.Fatalf("GenerateLLVM returned error: %v", err)
	}
	for _, want := range []string{"elseif.then", "elseif.next", "if.end"} {
		if !strings.Contains(ll, want) {
			t.Fatalf("LLVM IR missing %q:\n%s", want, ll)
		}
	}
}

func TestElseIfConditionMustBeBool(t *testing.T) {
	_, err := ParseAndCheckSource(strings.Join([]string{
		`if true {`,
		`println("ok")`,
		`} elseif 1 {`,
		`println("bad")`,
		`}`,
	}, "\n"))
	if err == nil {
		t.Fatal("expected elseif condition type error")
	}
	if !strings.Contains(err.Error(), "elseif condition must be bool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPrintAndLexingFeatures(t *testing.T) {
	src := strings.Join([]string{
		`/* multiline`,
		`   comment */`,
		`var x:int=1`,
		`var y:int=2`,
		`var s:string="a\n\tb"`,
		`print(s)`,
		`println(to_str(x!=y))`,
		`prinln("alias")`,
	}, "\n")
	prog, err := ParseAndCheckSource(src)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		t.Fatalf("GenerateC returned error: %v", err)
	}
	for _, want := range []string{`fnl_print(s);`, `fnl_println(fnl_str_bool((x != y)));`, `fnl_strdup("a\n\tb")`} {
		if !strings.Contains(csrc, want) {
			t.Fatalf("generated C missing %q:\n%s", want, csrc)
		}
	}
}
