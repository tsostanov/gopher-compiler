package main

import (
	"comp/internal/ast"
	"comp/internal/driver"
	"comp/internal/lexer"
	"comp/internal/options"
	"comp/internal/parser"
	"fmt"
	"os"
)

func main() {
	filePath, compatLoginov, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	input, err := driver.ReadInput(filePath, "var x: int = 123; print x + 5;")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	parse := parser.NewParserWithOptions(tokens, options.Mode{CompatLoginov: compatLoginov})
	statements, errors := parse.ParseWithRecovery()
	if len(statements) > 0 {
		printer := ast.NewAstPrinter()
		fmt.Print(printer.Print(statements))
	}
	for _, parseErr := range errors {
		fmt.Fprintln(os.Stderr, parseErr)
	}
	if len(errors) > 0 {
		os.Exit(1)
	}
}

func parseArgs(args []string) (string, bool, error) {
	var filePath string
	var compatLoginov bool

	for _, arg := range args {
		switch arg {
		case "--compat-loginov":
			compatLoginov = true
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return "", false, fmt.Errorf("unknown option: %s", arg)
			}
			if filePath != "" {
				return "", false, fmt.Errorf("multiple input files are not supported")
			}
			filePath = arg
		}
	}

	return filePath, compatLoginov, nil
}
