# FNL

FNL is Fazan's Neat Language. This repository contains `fnlc`, a small compiler written in Go with a handwritten lexer, recursive descent parser, semantic checker, AST JSON import/export, C backend, and LLVM IR emitter.

Current version: `v0.1`.

## License

This software is distributed under the BSD 3-Clause License. See [LICENSE](LICENSE).

## Build

```powershell
go build -o fnlc.exe .
```

## VS Code Syntax Highlighting

This repository includes a small local VS Code extension in `vscode-fnl` for `.fnl` syntax highlighting.

Install it for local development:

```powershell
Copy-Item -Recurse -Force .\vscode-fnl "$env:USERPROFILE\.vscode\extensions\fnl-syntax-0.0.1"
```

After installation, reopen any `.fnl` file or run `Developer: Reload Window` in VS Code.

## Usage

```powershell
fnlc.exe <source.fnl|source.fnl.ast> [-o source.exe] [--emit-c] [--emit-llvm] [--emit-ast] [--emit-ast-graph] [--backend=gcc|clang]
```

Defaults:

- Output executable: same base name as the source file, created in the current directory
- C emission: off
- LLVM IR emission: off
- AST JSON emission: off
- AST graph emission: off
- Backend: `gcc`
- Target: the platform `fnlc` is running on

Examples:

```powershell
go run . examples/hello.fnl
go run . examples/basics.fnl -o outputs/basics.exe --emit-c --emit-llvm --emit-ast --emit-ast-graph
go run . outputs/basics.fnl.ast -o outputs/basics_from_ast.exe
go run . examples/basics.fnl --backend=clang
```

`--emit-ast` writes a JSON AST file beside the output stem using the `.fnl.ast` extension. The AST JSON root includes `format`, `version`, and `kind` fields, and each AST node includes a `kind` field. `.fnl.ast` files can be fed back into `fnlc` directly and still go through semantic checking before code generation.

`--emit-ast-graph` writes a Graphviz DOT AST graph beside the output stem using the `.fnl.ast.dot` extension. It is independent of `--emit-ast`. You can render it with Graphviz:

```powershell
dot -Tsvg outputs/basics.fnl.ast.dot -o outputs/basics.fnl.ast.svg
dot -Tpng outputs/basics.fnl.ast.dot -o outputs/basics.fnl.ast.png
```

Cross-compilation is intentionally not supported. Build FNL programs on the platform you want to run them on, or emit C/LLVM IR and move that output to the target machine.

Emitted LLVM IR includes the FNL runtime helpers and can be built directly with Clang:

```powershell
clang outputs/basics.ll -o outputs/basics_from_ll.exe
```

On Windows, use a Clang environment with a configured linker/toolchain. With MinGW-style Clang, you may need:

```powershell
clang --target=x86_64-w64-windows-gnu outputs/basics.ll -o outputs/basics_from_ll.exe
```

## Language

```text
var name:string="FNL"
var x:int=2
var y:int=5
var ok:bool=x<y

println("hello " + name)

if ok {
    println(to_str(x+y))
} elseif x==0 {
    println("zero")
}

while x<10 {
    x=x+1
    if x==8 {
        break
    }
}

exit(0)
```

Types:

- `int`
- `double`
- `bool`
- `string`

Operators:

- Numeric: `+`, `-`, `*`, `/`
- Integer only: `%`
- Power: `^`
- Comparison: `<`, `<=`, `>`, `>=`, `==`
- Not equal: `!=`
- String concatenation: `string + string`
- String comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`

Rules:

- `if`, `elseif`, and `while` conditions must be `bool`.
- `elseif` branches are optional and may appear before `else`.
- `else` is optional.
- `print()` accepts only `string` and does not add a trailing newline.
- `println()` accepts only `string` and adds a trailing newline. `prinln()` is also accepted as an alias.
- `to_str()` converts `int`, `double`, `bool`, or `string` to `string`.
- `is_int()` and `to_int()` support `string` and `double` inputs.
- `is_double()` and `to_double()` support `string` and `int` inputs.
- `input()` waits for Enter-terminated stdin input and returns it as a `string`.
- `break` exits the nearest enclosing `while` loop.
- `exit(int)` terminates the process and returns the code to the OS.
- Strings support `\n` and `\t` escapes.
- Multiline comments use `/* ... */`.
- There are no implicit string conversions.
- Strings are immutable UTF-8 text. The runtime represents them internally with a data buffer and byte length; null termination is not the source of truth.
- On Windows, generated C executables print UTF-8 strings to console output through the wide console API when stdout is a terminal.
- String equality and ordering compare exact Unicode code point sequences. No locale collation, case folding, or Unicode normalization is performed.
- Math on booleans and strings is not allowed.
- Variable declarations and assignments require an exact type match.
- Mixed `int`/`double` numeric expressions promote to `double` only when the expression itself contains both numeric types.
- Numeric comparisons may also promote mixed `int`/`double` operands for the comparison.
- `^` follows numeric promotion: `int ^ int` returns `int`; any `double` operand returns `double`.
- `int` arithmetic is currently unchecked. Overflow, including overflow in `int ^ int`, follows the generated C backend behavior and should not be relied on.

Explicit conversion policy:

FNL keeps storage strict: variable declarations and assignments require an exact type match. Mixed numeric expressions may still promote `int` to `double` when an operator expression itself needs that promotion. General conversions use explicit `to_...` functions. Guard functions are used by the programmer before conversions that may fail or lose validity.

| Source | Destination | Function | Guard | Behavior |
| --- | --- | --- | --- | --- |
| `int` | `double` | `to_double(int)` | `is_double(int)` | Produces a finite `double`. Very large integers may lose integer precision because `double` cannot exactly represent every 64-bit integer. The guard returns whether the conversion can be exact and is only needed when exactness matters. |
| `string` | `double` | `to_double(string)` | `is_double(string)` | Parses a string as a `double`. Use the guard before converting untrusted text. Unguarded invalid input is programmer error. |
| `double` | `int` | `to_int(double)` | `is_int(double)` | Truncates toward zero. The guard should check that the value is finite and that the truncated result fits in `int`. Out-of-range values are unchecked programmer error. |
| `string` | `int` | `to_int(string)` | `is_int(string)` | Parses a base-10 signed integer string. Use the guard before converting untrusted text. Unguarded invalid or out-of-range input is programmer error. |
| `int` | `string` | `to_str(int)` | none | Converts to a signed decimal string. |
| `double` | `string` | `to_str(double)` | none | Converts to a simple human-readable decimal string suitable for display. Exact digit count, trailing zero behavior, and scientific notation are not guaranteed yet. |

Boolean parsing and numeric conversions are intentionally left out for now. `to_str(bool)` remains available for display.

Naming convention:

- Use `lower_snake_case` for variables and built-in function names.
- Type names and keywords are lowercase.
