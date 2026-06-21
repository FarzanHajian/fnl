# FNL

FNL is Fazan's Neat Language. This repository contains `fnlc`, a small compiler written in Go with a handwritten lexer, recursive descent parser, semantic checker, AST JSON import/export, C backend, and LLVM IR emitter.

## License

This software is distributed under the BSD 3-Clause License. See [LICENSE](LICENSE).

## Build

```powershell
go build -o fnlc.exe .
```

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
var x:int64=2
var y:int64=5
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

- `int64`
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
- String equality: `string == string`, `string != string`

Rules:

- `if`, `elseif`, and `while` conditions must be `bool`.
- `elseif` branches are optional and may appear before `else`.
- `else` is optional.
- `print()` accepts only `string` and does not add a trailing newline.
- `println()` accepts only `string` and adds a trailing newline. `prinln()` is also accepted as an alias.
- `to_str()` converts `int64`, `double`, `bool`, or `string` to `string`.
- `is_int64(string)` returns whether a string can be parsed as an `int64`.
- `to_int64(string)` parses a string as `int64`, returning `0` if invalid. Use `is_int64()` before converting user input.
- `is_double(string)` returns whether a string can be parsed as a `double`.
- `to_double(string)` parses a string as `double`, returning `0.0` if invalid. Use `is_double()` before converting user input.
- `input()` waits for Enter-terminated stdin input and returns it as a `string`.
- `break` exits the nearest enclosing `while` loop.
- `exit(int64)` terminates the process and returns the code to the OS.
- Strings support `\n` and `\t` escapes.
- Multiline comments use `/* ... */`.
- There are no implicit string conversions.
- Strings support equality and inequality comparisons, but not ordering comparisons.
- Math on booleans and strings is not allowed.
- `^` follows numeric promotion: `int64 ^ int64` returns `int64`; any `double` operand returns `double`.
- `int64` arithmetic is currently unchecked. Overflow, including overflow in `int64 ^ int64`, follows the generated C backend behavior and should not be relied on.

Naming convention:

- Use `lower_snake_case` for variables and built-in function names.
- Type names and keywords are lowercase.
