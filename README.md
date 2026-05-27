# comp

Small educational compiler front-end in Go. It includes:

- a lexer;
- a parser that builds an AST;
- a text AST printer;
- a Mermaid AST generator;
- a semantic analyzer with scope, initialization, and type checks;
- an AST optimizer with constant folding;
- an AST-based interpreter/executor.

## Project Layout

- `cmd/comp` - main CLI entry point
- `cmd/lexer-demo` - lab-style lexer demo
- `cmd/parser-demo` - lab-style parser demo with recovery
- `cmd/semantic-demo` - lab-style semantic diagnostics demo
- `cmd/interpreter-demo` - lab-style interpreter demo
- `internal/token` - token types and token metadata
- `internal/lexer` - lexical analysis
- `internal/parser` - syntax analysis
- `internal/ast` - AST nodes and AST renderers
- `internal/semantic` - semantic analysis
- `internal/optimizer` - AST optimization and verification
- `internal/executor` - AST execution/runtime
- `examples` - sample input program

## Pipeline

The default execution pipeline is:

```txt
Lexer -> Parser -> Semantic Analyzer -> Optimizer -> Interpreter
```

## Language Features

The language supports:

- variable declarations: `var x: int;`, `var x: int = 10;`, `var x = 10;`
- function declarations: `func add(a: int, b: int): int { return a + b; }`
- function calls: `add(1, 2)`
- assignment: `x = 20;`
- array literals and indexing: `[1, 2, 3]`, `xs[0]`, `xs[1] = 10`
- return statements: `return x;`, `return;`
- output: `print x;`
- blocks: `{ ... }`
- conditions: `if (...) ... else ...`
- loops: `while (...) ...`
- literals: integers, strings, booleans (`true`, `false`)
- operators: `+`, `-`, `*`, `/`, `!`, `==`, `!=`, `<`, `<=`, `>`, `>=`, `and`, `or`, `&&`, `||`
- line comments: `// comment`

## Type System

The default mode is statically and strictly typed.

Available types:

- `int`
- `bool`
- `string`
- array types such as `int[]`, `bool[]`, `string[]`

Strict-mode rules:

- a variable must have an explicit type annotation or an initializer
- if the type is omitted, it is inferred from the initializer
- assignments must match the declared or inferred type
- function arguments must match parameter types
- return expressions must match the declared function return type
- `if` and `while` conditions must be `bool`
- arithmetic operators work with `int`
- logical operators work with `bool`
- `+` supports `int + int` and `string + string`
- equality operators require both operands to have the same type

## Semantic Checks

The analyzer reports:

- use of an undeclared variable
- assignment to an undeclared variable
- use before initialization
- redeclaration in the same scope
- unused variables
- type mismatches in declarations and assignments
- invalid function calls and return types
- invalid operand types in expressions
- dead-code warnings for obviously false `if` and `while` conditions

## Run

Requirements: Go 1.25+.

Run with a file:

```bash
go run ./cmd/comp examples/program.txt
```

If no file path is provided, the built-in sample is used:

```txt
var x: int = 123; print x + 5;
```

CLI modes:

```bash
go run ./cmd/comp --tokens examples/program.txt
go run ./cmd/comp --ast examples/program.txt
go run ./cmd/comp --optimized-ast examples/program.txt
go run ./cmd/comp --ast-before-after examples/program.txt
go run ./cmd/comp --mermaid examples/program.txt
go run ./cmd/comp --semantic examples/program.txt
go run ./cmd/comp --verify-optimization examples/program.txt
go run ./cmd/comp --run examples/program.txt
```

`--run` executes the optimized AST.

`--ast` prints the parsed AST before optimization.

`--optimized-ast` prints the AST after optimization.

`--ast-before-after` prints both versions.

`--verify-optimization` executes both ASTs in isolation and checks that output and runtime behavior match.

## Lab-Style Demos

Lexer demo:

```bash
go run ./cmd/lexer-demo examples/program.txt
```

Parser demo:

```bash
go run ./cmd/parser-demo examples/program.txt
```

Semantic demo:

```bash
go run ./cmd/semantic-demo examples/program.txt
```

Interpreter demo:

```bash
go run ./cmd/interpreter-demo examples/program.txt
```

## Compatibility Mode

The compiler supports both `func` and `fun`.
`fun` is provided for compatibility with CompilerLabs-style examples.

With `--compat-loginov`, the compiler also accepts:

- untyped parameters: `fun add(a, b) { ... }`
- optional function return type
- `var x;`
- `return;`

## Example Program

`examples/program.txt`:

```txt
var numbers: int[] = [1 + 2, 4 * 2, 10 - 3];
print numbers[0];

numbers[1] = numbers[0] + numbers[2];
print numbers[1];

numbers = [8, 13, 21, 30 + 4];
print numbers[3];

var words: string[] = ["com" + "pi", "ler"];
words[0] = words[0] + words[1];
print words[0];

var flags: bool[] = [true, false, 10 > 3];
print flags[2];
```

## Tests

```bash
go test ./...
```
