# FNL Compiler: A Casual R&D Journal

## Overview

FNL, short for "Fazan's Neat Language", started as a small experiment: build a compiler for a simple programming language in Go. The first goal was modest but concrete: parse a tiny language with arithmetic, variables, `print`, and `if/else`, then produce a real executable.

The project has since grown into a compact but fairly complete learning compiler. It has a handwritten lexer, a recursive descent parser, a semantic checker, AST JSON import/export, Graphviz DOT AST graph export, a C backend for executable generation, and an LLVM IR emitter that now embeds the FNL runtime helpers so generated `.ll` files can be built directly with Clang.

This document summarizes the design journey, changes we made, mistakes we hit, and useful lessons learned along the way.

## Starting Point

The original language requirements were intentionally simple:

- Arithmetic: `+`, `-`, `*`, `/`, parentheses
- Variable declarations with `int` and `double`
- Variable usage and assignment
- Newline-terminated statements
- Built-in `print()`
- `if/else`
- A recursive descent parser
- A compiler written in Go
- Windows x64 executable output

The first implementation used Go for the compiler frontend and generated C as the backend output. Then it invoked GCC to produce a Windows executable.

This was a pragmatic shortcut. The frontend was a real compiler frontend, but the backend was really "FNL to C to executable". That led to the first joke of the project: "you cheater, you used GCC to compile." Fair. Accurate. Also useful.

## Early Language Design

At first, `if/else` used a newline-based form:

```fnl
if condition
print("yes")
else
print("no")
```

That quickly created ambiguity. Without an `end` keyword or braces, the parser had to guess where a block stopped. We discussed adding braces and agreed it was the right direction.

The language switched to:

```fnl
if condition {
print("yes")
} else {
print("no")
}
```

This solved the block boundary problem and made nested `if` statements natural.

## From TinyWin To FNL

After accidentally removing the earlier code, we regenerated the compiler from scratch and renamed the language to FNL.

The compiler became:

```text
fnlc.exe
```

The source extension became:

```text
.fnl
```

The project was also reorganized into multiple Go files instead of one large `main.go`. That was an explicit design request and a good one. The current layout is easier to work with:

- `token.go`: token definitions and keywords
- `lexer.go`: lexer
- `ast.go`: AST nodes
- `astjson.go`: AST JSON import/export
- `astgraph.go`: Graphviz DOT AST graph export
- `parser.go`: recursive descent parser
- `checker.go`: semantic checker
- `cgen.go`: C backend
- `llvmgen.go`: LLVM IR emitter
- `cli.go`: CLI parsing
- `target.go`: host/backend argument helpers
- `main.go`: orchestration
- `compiler_test.go`: regression tests

## Parser Design

The parser is still handwritten recursive descent.

Expression parsing is split by precedence:

```text
parseExpression
parseComparison
parseAdditive
parseMultiplicative
parsePower
parseUnary
parsePrimary
```

This keeps the grammar readable and makes it easy to add operators. The power operator `^` is right-associative, so:

```fnl
2^3^2
```

parses as:

```text
2^(3^2)
```

## Type System Growth

The language grew beyond `int` and `double`.

Current types:

- `int`
- `double`
- `bool`
- `string`

We added:

- Boolean literals: `true`, `false`
- Comparisons: `<`, `<=`, `>`, `>=`, `==`, `!=`
- Optional `else`
- `elseif`
- `while`
- `break`
- `exit()`
- `%`
- `^`
- `to_str()`
- `is_int()`
- `to_int()`
- `is_double()`
- `to_double()`
- `input()`
- `print()`
- `println()`
- `prinln()` as an accepted alias
- Multiline comments
- String escapes

Important type rules:

- `if`, `elseif`, and `while` conditions must be `bool`.
- `print`, `println`, and `prinln` accept only `string`.
- `print()` does not append a newline.
- `println()` and `prinln()` append a newline.
- `to_str()` converts values to strings.
- `is_int()` validates strings before integer conversion.
- `to_int()` converts valid integer strings to `int`, returning `0` for invalid input.
- `is_double()` validates strings before double conversion.
- `to_double()` converts valid double strings to `double`, returning `0.0` for invalid input.
- `input()` reads an Enter-terminated line from stdin and returns it as a string.
- `break` exits the nearest enclosing loop and is only valid inside `while`.
- `exit(int)` terminates the program and returns the code to the OS.
- `string + string` concatenates.
- There are no implicit string conversions.
- `%` is only valid for `int % int`.
- `^` follows numeric promotion: `int ^ int` returns `int`; any `double` operand returns `double`.
- `int` arithmetic is currently unchecked. Overflow follows the generated backend behavior and should not be relied on.
- Strings support `==`, `!=`, `<`, `<=`, `>`, and `>=`.
- String comparisons use exact Unicode code point sequence ordering, with no locale collation, case folding, or Unicode normalization.
- Bool supports `==` and `!=`, but not ordering comparisons.

## Strings And Printing

Strings were one of the biggest steps from "calculator language" toward something pleasant to use.

We decided that `print()` should only accept strings. That forced explicit conversion:

```fnl
println("x = " + to_str(x))
```

This avoided hidden conversions and kept the language rules simple.

Later, `print()` was changed to not add a newline, and `println()` was added for newline printing:

```fnl
print("A")
println("B")
```

prints:

```text
AB
```

followed by a newline.

String literals support:

```text
\n
\t
\"
\\
```

We also settled the v0 string model after a longer Unicode detour. FNL strings are immutable UTF-8 text. Under the hood, the C runtime represents a string as a small structure containing a data buffer and an authoritative byte length. A trailing null byte may exist for C interop, but null termination is not the string model.

The naming convention for future string helpers is:

```text
str_len(text)   number of Unicode code points
str_size(text)  number of UTF-8 bytes
```

For now, string ordering is available through operators only:

```fnl
"abc" < "abd"
"same" == "same"
```

The comparison is deterministic Unicode code point sequence ordering, not locale-aware human sorting.

## Comments

We added multiline comments:

```fnl
/* this is
   a multiline comment */
```

The lexer skips these before tokenizing operators, so `/* ... */` is recognized as a comment instead of `/` and `*` tokens.

## Backend Strategy

The executable backend currently generates C and invokes a system compiler.

The original backend was GCC-only. Later, the CLI allowed:

```text
--backend=gcc
--backend=clang
```

The default is still GCC.

The compiler can also emit LLVM IR:

```text
--emit-llvm
```

Executable generation by `fnlc` still goes through generated C, but emitted LLVM IR is no longer just for inspection. The `.ll` file embeds FNL runtime helper definitions and can be built directly with Clang when the host linker/C runtime environment is configured.

For example:

```powershell
fnlc.exe examples\basics.fnl -o outputs\basics.exe --emit-llvm
clang --target=x86_64-w64-windows-gnu outputs\basics.ll -o outputs\basics_from_ll.exe
```

On Windows, plain `clang outputs\basics.ll ...` may try to use the MSVC linker and fail unless the MSVC command prompt/environment is configured. The MinGW-style target worked on the development machine.

The compiler can also emit AST JSON:

```text
--emit-ast
```

This writes a `.fnl.ast` file beside the output stem. The file is JSON with a root like:

```json
{
  "format": "fnl.ast",
  "version": 1,
  "kind": "Program"
}
```

Each node has an explicit `kind` field. The compiler also accepts `.fnl.ast` files as input, skipping lexing/parsing and then running the semantic checker before code generation.

For a more visual view, the compiler can emit a Graphviz DOT AST graph:

```text
--emit-ast-graph
```

This writes a `.fnl.ast.dot` file. It is independent of `--emit-ast`. Graphviz can render it to SVG or PNG:

```powershell
dot -Tsvg outputs\basics.fnl.ast.dot -o outputs\basics.fnl.ast.svg
```

## CLI Evolution

The current CLI is:

```text
fnlc.exe <source.fnl|source.fnl.ast> [-o source.exe] [--emit-c] [--emit-llvm] [--emit-ast] [--emit-ast-graph] [--backend=gcc|clang]
```

We deliberately kept only one short option:

```text
-o
```

Everything else is a long option.

The output executable defaults to the source base name in the current directory. For example:

```powershell
fnlc.exe examples\fib.fnl
```

creates:

```text
fib.exe
```

on Windows.

## Cross-Compilation Detour

At one point, the compiler had:

```text
--target=win-x64
--target=linux-x64
```

This looked nice in the CLI, but it caused a real problem. Running on Windows with:

```powershell
fnlc.exe --target=linux-x64 examples\fib.fnl
```

still invoked the host Windows GCC, which produced a Windows binary. The filename changed, but the actual binary target did not. That was misleading.

We first added checks for missing cross-compilers, but then decided the whole feature was not worth keeping yet.

The final decision:

- No cross-compilation support.
- `fnlc` builds for the platform it is running on.
- To build Linux binaries, run `fnlc` on Linux.
- To move intermediate output, use `--emit-c` or `--emit-llvm`.

This made the tool more honest.

## Clang On Windows

Clang on Windows produced a few useful lessons.

First, plain `clang` on Windows often defaults to MSVC mode. Without an MSVC developer environment, it failed with:

```text
fatal error: 'math.h' file not found
```

We briefly made Clang use the MinGW target automatically:

```text
--target=x86_64-w64-windows-gnu
```

That worked on the current machine, but it changed the meaning of `--backend=clang`. We decided that if the user asks for Clang on Windows, it should use the normal MSVC-style Clang environment, not silently switch to MinGW.

So we reverted that and added a preflight check instead. On Windows, `--backend=clang` now checks for signs of a configured MSVC toolchain.

Then, inside a real MSVC command prompt, another issue appeared:

```text
lld-link: error: could not open 'm.lib'
```

The problem was `-lm`. That is the Unix/MinGW math library flag. MSVC does not use a separate `libm`; math functions are provided by the MSVC C runtime.

Final behavior:

- GCC still receives `-lm`.
- Clang on Windows does not receive `-lm`.
- Clang on Windows requires a configured MSVC environment.

## Scope Rules

Variables are block scoped.

Every brace block creates a new scope:

```fnl
while x<10 {
var y:int=1
println(to_str(y))
}
```

`y` is visible only inside the loop body.

Outer variables are visible inside inner blocks:

```fnl
var x:int=0

while x<10 {
x=x+1
}
```

This works because `x` is declared in the outer scope and assigned inside the loop.

## AST And Symbol Tables

As the AST export feature made clear, the AST can get large quickly. The AST is the compiler's tree-shaped representation of the program's syntax:

```text
Program
  VarDecl x
  While
    condition
    body
```

A symbol table is different. It is the compiler's record of names and their meanings. For the current FNL compiler, the important question is usually:

```text
what variables are currently declared, and what type does each one have?
```

The semantic checker currently has a simple symbol table stack:

```go
scopes []map[string]Type
```

Each map represents one scope. Entering a block pushes a new map; leaving the block pops it. When the checker sees:

```fnl
var x:int=1
```

it records:

```text
x -> int
```

When it later sees:

```fnl
x=x+1
```

it looks up `x` in the active scope stack to verify that `x` exists and to learn that it is an `int`.

So the distinction is:

```text
AST          = structure of the program
symbol table = known declarations and their meanings
```

They work together. The checker walks the AST and builds/queries the symbol table as it goes. Once FNL grows functions, user-defined types, arrays, structs, and modules, the symbol table will become more important and will likely split into separate tables for variables, functions, and types.

## Editor Tooling And Debugging

Once the compiler could emit executables, C, LLVM IR, AST JSON, and Graphviz DOT, the project started to feel less like a toy parser and more like a small programming environment. That naturally led to two usability questions: how pleasant is it to write FNL, and how painful is it to debug FNL?

The first answer was syntax highlighting. We added a small local VS Code extension under:

```text
vscode-fnl/
```

It is a TextMate grammar extension, not a full language server. That is the right size for the current project. It recognizes `.fnl` files and highlights:

- keywords such as `var`, `if`, `elseif`, `else`, `while`, `break`, and `exit`
- primitive types such as `int`, `double`, `bool`, and `string`
- built-ins such as `print`, `println`, `input`, `to_str`, `is_int`, `to_int`, `is_double`, and `to_double`
- strings and supported escape sequences
- multiline comments
- numbers, operators, and identifiers

This is a modest feature, but it changes the feeling of the language. FNL source files now look intentional in an editor instead of being anonymous plain text.

Debugging is trickier. The current compiler pipeline is:

```text
FNL source -> generated C -> native compiler -> executable
```

So when something goes wrong at runtime, the available debugger naturally sees C code, C line numbers, and generated C expressions. That is why debugging `sqrt.fnl` required looking at the emitted C. It worked, but it was not really FNL-level debugging.

We decided not to jump directly into a full debugger. The realistic roadmap starts with a compiler flag:

```text
--debug
```

The first version of `--debug` should keep the generated C file, compile with debug symbols, disable optimization, and add C `#line` directives. A generated statement could look conceptually like this:

```c
#line 42 "examples/sqrt.fnl"
guess = ((guess + ((double)num / guess)) / 2.0);
```

That would let GCC, Clang, GDB, LLDB, or Visual Studio map generated C instructions back to `.fnl` source locations where the native toolchain supports it. It would not be a perfect FNL debugger, but it would be a very practical bridge.

We also parked several later ideas:

- `--debug-map` for emitting a JSON map from FNL source ranges to generated C ranges
- `--trace` for compiler-injected runtime traces of statement execution and condition results
- `--trace-vars` for tracing variable values after declarations and assignments
- more readable generated C in debug mode, with stable variable names and source-line comments
- a future VS Code Debug Adapter Protocol integration on top of GDB, LLDB, or Visual Studio debugging

The important lesson is that debugging is not a single feature. It is a ladder. At the bottom is readable emitted code. Above that are source mappings and debug compiler flags. Much later comes a real source-level debugger that understands FNL variables and runtime values directly.

## Bugs We Found

### Optional Else Consumed Too Much

After making `else` optional, this program failed:

```fnl
if x==0 {
println("zero")
}
while x<2 {
x=x+1
}
```

The parser was looking ahead for `else` by consuming newlines after the `}`. If no `else` existed, it had already consumed the newline separating the `if` from the `while`. The outer parser then complained:

```text
expected newline after statement
```

Fix: save the parser position before looking for `else`, and restore it if no `else` is found.

### Temporary C Files Interfered With Go Build

Generated temporary C files were initially created near the output path. If the output path was the project root, a temporary `.c` file could briefly appear in the Go package directory. Then:

```powershell
go build
```

could fail because Go saw an unexpected C source file.

Fix: temporary C files are now created in the OS temp directory unless `--emit-c` is requested.

### LLVM String Constants Were Emitted Inside Main

At one point, string constants in emitted LLVM IR appeared inside `main`, which is invalid LLVM IR.

Fix: LLVM string constants are now collected and emitted at module scope before function definitions.

### `print()` Newline Behavior

Originally, `print()` always added a newline. Later we changed the language:

- `print()` means no newline
- `println()` means newline

This required changes in:

- parser/AST
- checker
- C backend
- LLVM IR emitter
- tests
- examples

## Current Example

The Fibonacci example demonstrates variables, loops, conditionals, string conversion, and printing:

```fnl
var total_terms:int = 20
var first:int = 0
var second:int = 1

var current_term:int = 1
println("Here are the first " + to_str(total_terms) + " terms of the Fibonacci Serie:")
print(to_str(first))

if total_terms > 1 {
    current_term = 2
    print(", ")
    print(to_str(second))
}

while current_term < total_terms {
    current_term = current_term + 1
    second = first + second
    first = second - first
    print(", ")
    print(to_str(second))
}
```

## Testing

The project has regression tests for:

- parsing
- semantic type errors
- generated C content
- generated LLVM IR content
- AST JSON export/import
- Graphviz DOT AST graph export
- optional `else`
- `elseif`
- multiline comments
- string escapes
- `!=`
- `print` vs `println`
- `input()`
- `break` and `exit()`
- CLI behavior
- backend argument rules

Run tests with:

```powershell
go test ./...
```

## Current Status

FNL `v0.1` is a small, typed, block-scoped language compiled through generated C by default.

The `v0.1` label marks the first named snapshot after the UTF-8 string model and string ordering rules were implemented and documented.

It is not yet a native machine-code compiler, but it has a strong compiler-shaped architecture:

- lexer
- parser
- AST
- semantic checker
- code generators
- CLI
- tests

The next natural improvements would be:

- single-line comments
- `for` loops
- functions
- return statements
- arrays
- richer string operations such as ordering or substring search
- better memory management for generated strings
- direct LLVM executable generation through `fnlc`, instead of only emitting buildable `.ll`
- better diagnostic spans and source snippets
- first-stage debug mode with `#line`, kept C output, and `-g -O0`
- richer editor tooling beyond basic VS Code syntax highlighting
- file I/O
- user-defined `type` declarations for arrays and structs
- a built-in linter

The project is still small enough to understand, which is good. It has grown just enough to expose real compiler design questions without becoming a maze.

## Naming Convention

We decided to standardize on `lower_snake_case` for FNL names. Variables and built-in functions should use that style:

```fnl
var user_input:string=input()
var parsed_value:int=to_int(user_input)
```

Keywords and type names remain lowercase. If constants are added later, `UPPER_SNAKE_CASE` is a likely convention for them.
