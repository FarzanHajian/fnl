# FNL Compiler Project Notes

These notes are a handoff for future Codex threads or contributors.

## Project

FNL means "Fazan's Neat Language". The compiler executable is `fnlc` / `fnlc.exe`, written in Go.

Source files use the `.fnl` extension.

Current project version: `v0.1`.

`v0.1` marks the first named snapshot after settling the core string model: immutable UTF-8 strings, length-aware C runtime representation, and string operators for concatenation, equality, inequality, and ordering.

## Current CLI

```text
fnlc.exe <source.fnl|source.fnl.ast> [-o source.exe] [--emit-c] [--emit-llvm] [--emit-ast] [--emit-ast-graph] [--backend=gcc|clang]
```

Rules:

- `-o` is the only short option.
- All other options are long `--...` options.
- Cross-compilation is intentionally not supported.
- The generated executable targets the platform where `fnlc` is running.
- Default backend is `gcc`.
- `--emit-c` writes generated C beside the executable output path.
- `--emit-llvm` writes generated LLVM IR beside the executable output path.
- `--emit-ast` writes AST JSON beside the executable output path using the `.fnl.ast` extension.
- `--emit-ast-graph` writes a Graphviz DOT AST graph beside the executable output path using the `.fnl.ast.dot` extension. It is independent of `--emit-ast`.
- `.fnl` inputs are lexed and parsed into an AST.
- `.fnl.ast` inputs are decoded from AST JSON and then semantically checked before code generation.

## Backend Notes

The compiler currently generates C for executable builds.

- `--backend=gcc` invokes GCC.
- `--backend=clang` invokes Clang.
- On Windows, Clang is expected to use an MSVC-configured environment. `fnlc` checks for common MSVC toolchain indicators before invoking Clang.
- On Windows with Clang/MSVC, `fnlc` does not pass `-lm`, because MSVC does not use a separate Unix-style `libm`.
- GCC still receives `-lm`.

LLVM IR emission embeds the FNL runtime helper definitions directly, similar to the C backend. Generated `.ll` files can be built directly with Clang when the host C runtime/linker toolchain is configured.

Example:

```powershell
clang outputs\basics.ll -o outputs\basics_from_ll.exe
```

On Windows with MinGW-style Clang, an explicit target may be needed:

```powershell
clang --target=x86_64-w64-windows-gnu outputs\basics.ll -o outputs\basics_from_ll.exe
```

## Architecture

The code is intentionally modular:

- `token.go`: token kinds and keywords
- `lexer.go`: lexer, string escapes, multiline comments
- `ast.go`: AST node structs
- `astjson.go`: stable AST JSON import/export
- `astgraph.go`: Graphviz DOT AST graph export
- `parser.go`: handwritten recursive descent parser
- `checker.go`: semantic/type checker
- `cgen.go`: C backend and embedded runtime helpers
- `llvmgen.go`: LLVM IR emitter
- `cli.go`: command-line parsing
- `target.go`: host-platform defaults and backend arguments
- `main.go`: CLI entry point and build orchestration
- `compiler_test.go`: regression tests
- `vscode-fnl/`: local VS Code TextMate grammar extension for `.fnl` syntax highlighting

## Parser

The parser is handwritten recursive descent. Expression precedence is split into parser functions:

```text
parseExpression
parseComparison
parseAdditive
parseMultiplicative
parsePower
parseUnary
parsePrimary
```

`^` is right-associative.

Blocks use braces:

```fnl
if condition {
println("yes")
} elseif other_condition {
println("maybe")
} else {
println("no")
}
```

`elseif` branches are optional and may appear before `else`. `else` is optional.

## Semantic Checking And Symbol Table

The AST records the source structure: statements, expressions, operators, literals, and source positions. It does not by itself answer questions such as "which variable does this name refer to?" or "what type is this expression?"

Those questions are handled after parsing by the semantic checker in `checker.go`. The checker maintains a symbol table as a stack of scopes:

```go
scopes []map[string]Type
```

Each `{ ... }` block pushes a new scope and pops it when the block ends. Lookup walks from the innermost scope outward, so inner declarations can see outer variables. Redeclaring the same variable in the same scope is rejected, while declaring the same name in an inner block is currently allowed as shadowing.

This distinction matters for the grammar: BNF and EBNF describe what token sequences are syntactically valid FNL source code. They do not describe type validity, variable lifetime, or name lookup rules. Those are semantic rules enforced after the AST exists.

## Language Features

Types:

- `int64`
- `double`
- `bool`
- `string`

Statements:

- variable declaration: `var name:type=expression`
- assignment: `name=expression`
- `if` with optional `elseif` branches and optional `else`
- `while`
- `break`
- `print(expression)`
- `println(expression)`
- `prinln(expression)` is accepted as an alias for `println`
- `exit(expression)` terminates the process with an OS exit code
- `input()` returns an Enter-terminated line from stdin as `string`

Comments:

- multiline comments: `/* ... */`

Strings:

- string literals use double quotes
- supported escapes include `\n`, `\t`, `\"`, and `\\`
- string values are immutable UTF-8 text
- the C runtime represents strings internally as a structure with a data buffer and authoritative byte length
- the runtime may keep a trailing null byte for C interop, but null termination does not define the string
- on Windows, generated C executables print UTF-8 strings through `WriteConsoleW` when stdout is a console and fall back to raw UTF-8 bytes for pipes/files

## Type Rules

- `if`, `elseif`, and `while` conditions must be `bool`.
- `print`, `println`, and `prinln` accept only `string`.
- `print()` does not add a trailing newline.
- `println()` and `prinln()` add a trailing newline.
- `to_str(expression)` converts any FNL value to `string`.
- `is_int64(string)` returns whether a string can be parsed as an `int64`.
- `to_int64(string)` parses a string as `int64`, returning `0` if invalid.
- `is_double(string)` returns whether a string can be parsed as a `double`.
- `to_double(string)` parses a string as `double`, returning `0.0` if invalid.
- `input()` waits for Enter-terminated stdin input and returns a `string`.
- `break` is valid only inside `while` and exits the nearest enclosing loop.
- `exit(int64)` terminates the process and returns the code to the OS.
- Bool string conversion returns lowercase `true` or `false`.
- `string + string` concatenates strings.
- No implicit string conversions are allowed.
- `string + int64` is invalid; use `string + to_str(int64Value)`.
- Math on booleans is invalid.
- Math on strings is invalid except for `string + string`.
- `int64` mixed with `double` in `+`, `-`, `*`, `/`, `^` returns `double`.
- `int64` with `int64` in `+`, `-`, `*`, `/`, `^` returns `int64`.
- `double` with `double` in `+`, `-`, `*`, `/`, `^` returns `double`.
- `%` is valid only for `int64 % int64`, returning `int64`.
- `^` follows numeric promotion and is right-associative.
- `int64` arithmetic is currently unchecked. Overflow, including overflow in `int64 ^ int64`, follows the generated C backend behavior and should not be relied on.
- Comparisons return `bool`.
- Comparison operands must have matching types.
- Strings support `==`, `!=`, `<`, `<=`, `>`, and `>=`.
- String equality and ordering compare exact Unicode code point sequences. No locale collation, case folding, or Unicode normalization is performed.
- String ordering is lexicographic; if one string is a prefix of another, the shorter string is less.
- Bool supports `==` and `!=`, but not ordering comparisons.

## Naming Convention

- Use `lower_snake_case` for variables and built-in function names.
- Type names and keywords are lowercase.
- Future user-defined functions should also use `lower_snake_case`.
- Future constants, if added, may use `UPPER_SNAKE_CASE`.
- Built-in and standard-library functions should prefer short, consistent prefixes rather than method-style names.
- Keep `to_` for explicit conversions, for example `to_str`, `to_int64`, and `to_double`.
- Keep `is_` for validation/predicate helpers, for example `is_int64` and `is_double`.
- Use domain prefixes for standard-library families, for example `str_` for string operations, `io_` for I/O, `fs_` for filesystem operations, `path_` for path manipulation, and `math_` for math helpers.
- Prefer `str_len(text)` over method or namespace forms like `string.len(text)` or `text.len()`.
- Prefer general conversion names such as `to_str(value)` over fully qualified input-type names such as `double_to_str(value)`.
- Future `str_len(text)` should return the number of Unicode code points.
- Future `str_size(text)` should return the number of UTF-8 bytes.

## Current Operators

Arithmetic:

```text
+ - * / % ^
```

Comparison:

```text
< <= > >= == !=
```

String:

```text
+ == != < <= > >=
```

## Grammar

This grammar describes `.fnl` source files. `.fnl.ast` JSON files and `.fnl.ast.dot` Graphviz DOT files are compiler artifact formats, not part of the FNL source grammar.

There are two grammar views:

- The language reference grammar describes the intended source language in a human-facing way.
- The parser implementation grammar describes the shape currently used by the handwritten recursive descent parser.

### Language Reference Grammar

The reference grammar intentionally keeps expressions compact. Operator precedence and associativity are specified after the grammar instead of being encoded as parser layers.

Precedence, highest to lowest:

```text
unary -
^                 right-associative
* / %
+ -
< <= > >= == !=
```

`+`, `-`, `*`, `/`, and `%` are left-associative. Comparison operators are non-associative; FNL currently allows one comparison operator per expression level, so `a < b < c` requires explicit rewriting.

BNF notation:

```bnf
<program> ::= <newlines-opt> <statement-list-opt> <newlines-opt> <eof>

<statement-list-opt> ::= ε
                       | <statement-list>

<statement-list> ::= <statement>
                   | <statement> <newlines> <statement-list>

<statement> ::= <var-declaration>
              | <assignment>
              | <print-statement>
              | <println-statement>
              | <exit-statement>
              | <break-statement>
              | <if-statement>
              | <while-statement>

<var-declaration> ::= "var" <identifier> ":" <type> "=" <expression>

<assignment> ::= <identifier> "=" <expression>

<print-statement> ::= "print" "(" <expression> ")"

<println-statement> ::= "println" "(" <expression> ")"
                      | "prinln" "(" <expression> ")"

<exit-statement> ::= "exit" "(" <expression> ")"

<break-statement> ::= "break"

<if-statement> ::= "if" <expression> <block> <elseif-list-opt> <else-opt>

<elseif-list-opt> ::= ε
                    | <elseif-list>

<elseif-list> ::= <elseif-clause>
                | <elseif-clause> <elseif-list>

<elseif-clause> ::= <newlines-opt> "elseif" <expression> <block>

<else-opt> ::= ε
             | <newlines-opt> "else" <block>

<while-statement> ::= "while" <expression> <block>

<block> ::= "{" <newlines-opt> <statement-list> <newlines-opt> "}"

<type> ::= "int64"
         | "double"
         | "bool"
         | "string"

<expression> ::= <unary-expression>
               | <expression> <binary-operator> <expression>

<unary-expression> ::= "-" <unary-expression>
                     | <primary>

<binary-operator> ::= "+"
                    | "-"
                    | "*"
                    | "/"
                    | "%"
                    | "^"
                    | "<"
                    | "<="
                    | ">"
                    | ">="
                    | "=="
                    | "!="

<primary> ::= <integer-literal>
            | <double-literal>
            | <bool-literal>
            | <string-literal>
            | <identifier>
            | <builtin-call>
            | "(" <expression> ")"

<builtin-call> ::= <to-str-call>
                 | <is-int64-call>
                 | <to-int64-call>
                 | <is-double-call>
                 | <to-double-call>
                 | <input-call>

<to-str-call> ::= "to_str" "(" <expression> ")"

<is-int64-call> ::= "is_int64" "(" <expression> ")"

<to-int64-call> ::= "to_int64" "(" <expression> ")"

<is-double-call> ::= "is_double" "(" <expression> ")"

<to-double-call> ::= "to_double" "(" <expression> ")"

<input-call> ::= "input" "(" ")"

<bool-literal> ::= "true"
                 | "false"

<string-literal> ::= "\"" <string-characters-opt> "\""

<string-characters-opt> ::= ε
                          | <string-character> <string-characters-opt>

<string-character> ::= <non-quote-non-backslash-non-newline-character>
                     | "\\" "n"
                     | "\\" "t"
                     | "\\" "\""
                     | "\\" "\\"

<newlines-opt> ::= ε
                 | <newlines>

<newlines> ::= <newline>
             | <newline> <newlines>

<identifier> ::= <letter-or-underscore> <identifier-tail-opt>

<identifier-tail-opt> ::= ε
                        | <identifier-char> <identifier-tail-opt>

<identifier-char> ::= <letter-or-underscore>
                    | <digit>

<integer-literal> ::= <digit> <digits-opt>

<double-literal> ::= <digit> <digits-opt> "." <digit> <digits-opt>

<digits-opt> ::= ε
               | <digit> <digits-opt>
```

EBNF notation:

```ebnf
program        = newlines? statementList? newlines? EOF ;

statementList  = statement (newlines statement)* ;

statement      = varDeclaration
               | assignment
               | printStatement
               | printlnStatement
               | exitStatement
               | breakStatement
               | ifStatement
               | whileStatement ;

varDeclaration = "var" identifier ":" type "=" expression ;

assignment     = identifier "=" expression ;

printStatement = "print" "(" expression ")" ;

printlnStatement = ("println" | "prinln") "(" expression ")" ;

exitStatement  = "exit" "(" expression ")" ;

breakStatement = "break" ;

ifStatement    = "if" expression block (newlines? "elseif" expression block)* (newlines? "else" block)? ;

whileStatement = "while" expression block ;

block          = "{" newlines? statementList newlines? "}" ;

type           = "int64" | "double" | "bool" | "string" ;

expression     = unaryExpression | expression binaryOperator expression ;

unaryExpression = "-" unaryExpression | primary ;

binaryOperator = "+" | "-" | "*" | "/" | "%" | "^" | "<" | "<=" | ">" | ">=" | "==" | "!=" ;

primary        = integerLiteral
               | doubleLiteral
               | boolLiteral
               | stringLiteral
               | identifier
               | builtinCall
               | "(" expression ")" ;

builtinCall    = toStrCall
               | isInt64Call
               | toInt64Call
               | isDoubleCall
               | toDoubleCall
               | inputCall ;

toStrCall      = "to_str" "(" expression ")" ;

isInt64Call    = "is_int64" "(" expression ")" ;

toInt64Call    = "to_int64" "(" expression ")" ;

isDoubleCall   = "is_double" "(" expression ")" ;

toDoubleCall   = "to_double" "(" expression ")" ;

inputCall      = "input" "(" ")" ;

boolLiteral    = "true" | "false" ;

stringLiteral  = "\"" stringCharacter* "\"" ;

stringCharacter = nonQuoteNonBackslashNonNewlineCharacter
                | "\\" "n"
                | "\\" "t"
                | "\\" "\""
                | "\\" "\\" ;

newlines       = newline+ ;

identifier     = letterOrUnderscore identifierChar* ;

identifierChar = letterOrUnderscore | digit ;

integerLiteral = digit+ ;

doubleLiteral  = digit+ "." digit+ ;
```

### Parser Implementation Grammar

The parser grammar removes left recursion and encodes precedence directly. This is the grammar shape to compare against `parser.go`. The BNF tail productions represent parser loops; `+`, `-`, `*`, `/`, and `%` still build left-associative ASTs.

BNF notation:

```bnf
<program> ::= <newlines-opt> <statement-list-opt> <newlines-opt> <eof>

<statement-list-opt> ::= ε
                       | <statement-list>

<statement-list> ::= <statement>
                   | <statement> <newlines> <statement-list>

<statement> ::= <var-declaration>
              | <assignment>
              | <print-statement>
              | <println-statement>
              | <exit-statement>
              | <break-statement>
              | <if-statement>
              | <while-statement>

<var-declaration> ::= "var" <identifier> ":" <type> "=" <expression>

<assignment> ::= <identifier> "=" <expression>

<print-statement> ::= "print" "(" <expression> ")"

<println-statement> ::= "println" "(" <expression> ")"
                      | "prinln" "(" <expression> ")"

<exit-statement> ::= "exit" "(" <expression> ")"

<break-statement> ::= "break"

<if-statement> ::= "if" <expression> <block> <elseif-list-opt> <else-opt>

<elseif-list-opt> ::= ε
                    | <elseif-list>

<elseif-list> ::= <elseif-clause>
                | <elseif-clause> <elseif-list>

<elseif-clause> ::= <newlines-opt> "elseif" <expression> <block>

<else-opt> ::= ε
             | <newlines-opt> "else" <block>

<while-statement> ::= "while" <expression> <block>

<block> ::= "{" <newlines-opt> <statement-list> <newlines-opt> "}"

<type> ::= "int64"
         | "double"
         | "bool"
         | "string"

<expression> ::= <comparison>

<comparison> ::= <additive> <comparison-tail-opt>

<comparison-tail-opt> ::= ε
                        | "<" <additive>
                        | "<=" <additive>
                        | ">" <additive>
                        | ">=" <additive>
                        | "==" <additive>
                        | "!=" <additive>

<additive> ::= <multiplicative> <additive-tail-opt>

<additive-tail-opt> ::= ε
                      | "+" <multiplicative> <additive-tail-opt>
                      | "-" <multiplicative> <additive-tail-opt>

<multiplicative> ::= <power> <multiplicative-tail-opt>

<multiplicative-tail-opt> ::= ε
                            | "*" <power> <multiplicative-tail-opt>
                            | "/" <power> <multiplicative-tail-opt>
                            | "%" <power> <multiplicative-tail-opt>

<power> ::= <unary>
          | <unary> "^" <power>

<unary> ::= "-" <unary>
          | <primary>

<primary> ::= <integer-literal>
            | <double-literal>
            | <bool-literal>
            | <string-literal>
            | <identifier>
            | <to-str-call>
            | <is-int64-call>
            | <to-int64-call>
            | <is-double-call>
            | <to-double-call>
            | <input-call>
            | "(" <expression> ")"

<to-str-call> ::= "to_str" "(" <expression> ")"

<is-int64-call> ::= "is_int64" "(" <expression> ")"

<to-int64-call> ::= "to_int64" "(" <expression> ")"

<is-double-call> ::= "is_double" "(" <expression> ")"

<to-double-call> ::= "to_double" "(" <expression> ")"

<input-call> ::= "input" "(" ")"

<bool-literal> ::= "true"
                 | "false"

<string-literal> ::= "\"" <string-characters-opt> "\""

<string-characters-opt> ::= ε
                          | <string-character> <string-characters-opt>

<string-character> ::= <non-quote-non-backslash-non-newline-character>
                     | "\\" "n"
                     | "\\" "t"
                     | "\\" "\""
                     | "\\" "\\"

<newlines-opt> ::= ε
                 | <newlines>

<newlines> ::= <newline>
             | <newline> <newlines>

<identifier> ::= <letter-or-underscore> <identifier-tail-opt>

<identifier-tail-opt> ::= ε
                        | <identifier-char> <identifier-tail-opt>

<identifier-char> ::= <letter-or-underscore>
                    | <digit>

<integer-literal> ::= <digit> <digits-opt>

<double-literal> ::= <digit> <digits-opt> "." <digit> <digits-opt>

<digits-opt> ::= ε
               | <digit> <digits-opt>
```

EBNF notation:

```ebnf
program        = newlines? statementList? newlines? EOF ;

statementList  = statement (newlines statement)* ;

statement      = varDeclaration
               | assignment
               | printStatement
               | printlnStatement
               | exitStatement
               | breakStatement
               | ifStatement
               | whileStatement ;

varDeclaration = "var" identifier ":" type "=" expression ;

assignment     = identifier "=" expression ;

printStatement = "print" "(" expression ")" ;

printlnStatement = ("println" | "prinln") "(" expression ")" ;

exitStatement = "exit" "(" expression ")" ;

breakStatement = "break" ;

ifStatement    = "if" expression block (newlines? "elseif" expression block)* (newlines? "else" block)? ;

whileStatement = "while" expression block ;

block          = "{" newlines? statementList newlines? "}" ;

type           = "int64"
               | "double"
               | "bool"
               | "string" ;

expression     = comparison ;

comparison     = additive (("<" | "<=" | ">" | ">=" | "==" | "!=") additive)? ;

additive       = multiplicative (("+" | "-") multiplicative)* ;

multiplicative = power (("*" | "/" | "%") power)* ;

power          = unary ("^" power)? ;

unary          = "-" unary
               | primary ;

primary        = integerLiteral
               | doubleLiteral
               | boolLiteral
               | stringLiteral
               | identifier
               | toStrCall
               | isInt64Call
               | toInt64Call
               | isDoubleCall
               | toDoubleCall
               | inputCall
               | "(" expression ")" ;

toStrCall      = "to_str" "(" expression ")" ;

isInt64Call    = "is_int64" "(" expression ")" ;

toInt64Call    = "to_int64" "(" expression ")" ;

isDoubleCall   = "is_double" "(" expression ")" ;

toDoubleCall   = "to_double" "(" expression ")" ;

inputCall      = "input" "(" ")" ;

boolLiteral    = "true" | "false" ;

stringLiteral  = "\"" stringCharacter* "\"" ;

stringCharacter = nonQuoteNonBackslashNonNewlineCharacter
                | "\\" "n"
                | "\\" "t"
                | "\\" "\""
                | "\\" "\\" ;

newlines       = newline+ ;

identifier     = letterOrUnderscore identifierChar* ;

identifierChar = letterOrUnderscore | digit ;

integerLiteral = digit+ ;

doubleLiteral  = digit+ "." digit+ ;
```

## Known Design Decisions

- The project no longer supports `--target`; cross-compilation was removed to avoid misleading results.
- The C backend is the executable backend for now.
- LLVM IR embeds the FNL runtime helpers and is intended to be directly buildable with Clang.
- AST JSON files use the `.fnl.ast` extension, include `format: "fnl.ast"`, `version: 1`, and explicit `kind` fields, and are accepted as compiler inputs.
- AST graph files use Graphviz DOT syntax and the `.fnl.ast.dot` extension.
- Variables are block scoped. Every `{ ... }` block creates a new scope.
- A variable declared inside a loop body is visible only inside that loop body.

## Future Design Notes

These are design ideas discussed but not implemented yet.

- Rename `int64` to `int`, with `int` still meaning a fixed signed 64-bit integer. Matching built-ins would likely become `is_int` and `to_int`.
- Keep conversion functions explicit for now. Type names should not become callable C-style casts yet, because FNL does not intend to support every conversion from every type to every other type.
- Keep `double`, `is_double`, and `to_double` for clarity unless the primitive type naming style changes more broadly.
- Consider adding `byte` as an unsigned 8-bit storage/data type for future files, streams, and binary buffers. Initial design preference: no normal arithmetic on `byte`; possible future bit operations include `&`, `|`, `xor`, `~`, `<<`, and `>>`.
- Widening arithmetic operators were discussed but deferred. For now, numeric operators use ordinary promotion rules. Future explicit widening may be handled with conversion built-ins or dedicated built-ins rather than symbolic operators like `**`.
- Add user-defined type declarations with the keyword `type`, not `compose`.

Potential type declaration syntax:

```fnl
type int_arr = int[10]

type my_struct = struct {
    name:string
    values:int_arr
}
```

- Arrays and structs are considered future "sophisticated" data types.
- The first version of user-defined types should probably require referenced types to be declared earlier in the file. Allowing forward references would require an additional type collection/resolution pass and cycle detection.
- The compiler already has phases: lexing, parsing, semantic checking, code generation, and backend invocation. User-defined types would likely add a more explicit type declaration pass before ordinary statement checking.
- A future bootstrapping path will need functions, return statements, arrays, structs, richer string operations, and file I/O before implementing the compiler in FNL becomes realistic.
- Future function declarations may follow the same declaration shape as `var` and `type`:

```fnl
func add = (a:int64, b:int64) int64 {
    return a+b
}

func say = (text:string) {
    println(text)
}
```

- In that design, a return type after the parameter list means the function returns a value. Omitting the return type means the function does not return a value. The `return` keyword remains a statement inside function bodies.
- A future standard library should be written mostly in FNL, but it needs functions first.
- Standard-library functions should call lower-level intrinsics for operations that ordinary FNL cannot express, such as raw printing, reading stdin, exiting the process, allocation, string internals, and platform APIs.
- Intrinsics should be treated as a private compiler/runtime boundary, not as the normal user-facing API.
- Current built-ins such as `print`, `println`, `input`, `to_str`, `is_int64`, `to_int64`, `is_double`, and `to_double` are effectively prelude-style functions implemented directly by the compiler/runtime for now.
- A future cleanup should replace parser-special built-in call nodes with a general `CallExpr`, then let semantic checking resolve calls against variables/functions/built-ins.
- The likely layering is: FNL source calls standard-library/prelude functions; standard-library FNL calls private intrinsics; backends lower intrinsics to C helpers, LLVM helpers, or platform-specific runtime code.
- Intrinsic naming should use a reserved internal prefix such as `__fnl_`, for example `__fnl_print_raw`, `__fnl_input_line`, `__fnl_exit`, `__fnl_str_concat`, and `__fnl_str_eq`.
- Standard-library naming should keep the prefix convention even after modules exist. Examples: `str_len`, `str_concat`, `str_compare`, `io_read_line`, `fs_exists`, and `math_sqrt`.
- A built-in linter is a desired future compiler feature.

Possible linter CLI:

```powershell
fnlc.exe source.fnl --lint
fnlc.exe source.fnl --lint-only
```

- `--lint` would compile normally but also report lint warnings.
- `--lint-only` would run lint checks without producing an executable.

Possible lint rules:

- enforce 4-space indentation for FNL source files
- reject or warn on tabs
- warn on trailing whitespace
- enforce `lower_snake_case` for variables, functions, and built-ins
- check spacing around operators and after commas
- check blank lines between top-level declarations once functions/types exist
- warn about unused variables
- warn about variable shadowing
- warn about unreachable statements after `break`, `exit`, and eventually `return`
- suggest `println()` instead of `print("...\n")`
- warn about obviously overflow-prone integer expressions when literals make that visible
- warn when `to_int64()` or `to_double()` are used without a nearby validating `is_int64()` or `is_double()` guard

Future debugging support should start small and build toward real source-level FNL debugging.

Possible debugging CLI:

```powershell
fnlc.exe source.fnl --debug
fnlc.exe source.fnl --debug-map
fnlc.exe source.fnl --trace
fnlc.exe source.fnl --trace-vars
```

- `--debug` should keep the generated C file, compile with debug symbols, disable optimization, and emit C `#line` directives that map generated C back to `.fnl` source lines.
- For GCC/Clang backends, `--debug` should add flags like `-g` and `-O0`.
- Generated C should remain readable in debug mode, with stable variable names and comments showing the original FNL source line where helpful.
- `#line` directives are the first practical bridge between native debuggers and FNL source locations. They should let tools like GDB, LLDB, or Visual Studio show `.fnl` file and line information where the C toolchain supports it.
- `--debug-map` could emit a JSON source map, for example `source.fnl.map`, recording FNL line/column ranges and generated C line ranges.
- `--trace` could inject runtime prints for statement execution, condition results, and loop progress.
- `--trace-vars` could extend tracing with variable values after declarations and assignments.
- Full VS Code debugging would likely require a Debug Adapter Protocol implementation on top of GDB, LLDB, or the Visual Studio debugger.
- A real FNL debugger will need reliable source maps, stable generated names, runtime metadata for strings and future compound types, and a way to translate native debugger values back into FNL values.

## Useful Commands

Run tests:

```powershell
go test ./...
```

Build compiler:

```powershell
go build -o outputs/fnlc.exe .
```

Compile sample:

```powershell
.\outputs\fnlc.exe examples\basics.fnl -o outputs\basics.exe --emit-c --emit-llvm
```

Emit AST JSON:

```powershell
.\outputs\fnlc.exe examples\basics.fnl -o outputs\basics.exe --emit-ast
```

Emit a Graphviz DOT AST graph:

```powershell
.\outputs\fnlc.exe examples\basics.fnl -o outputs\basics.exe --emit-ast-graph
```

Render the AST graph with Graphviz:

```powershell
dot -Tsvg outputs\basics.fnl.ast.dot -o outputs\basics.fnl.ast.svg
```

Compile from AST JSON:

```powershell
.\outputs\fnlc.exe outputs\basics.fnl.ast -o outputs\basics_from_ast.exe
```

Build emitted LLVM IR directly with Clang:

```powershell
clang --target=x86_64-w64-windows-gnu outputs\basics.ll -o outputs\basics_from_ll.exe
```

Compile interactive hello:

```powershell
.\outputs\fnlc.exe examples\hello.fnl -o outputs\hello.exe
```

Compile Fibonacci:

```powershell
.\outputs\fnlc.exe examples\fib.fnl -o outputs\fib.exe --emit-c --emit-llvm
```
