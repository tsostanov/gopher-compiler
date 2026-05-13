package main

import (
	"comp/internal/ast"
	"comp/internal/driver"
	"comp/internal/executor"
	"comp/internal/lexer"
	"comp/internal/options"
	"comp/internal/parser"
	"comp/internal/semantic"
	"fmt"
	"os"
)

func main() {
	cliOptions, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	input, err := driver.ReadInput(cliOptions.filePath, "var x: int = 123; print x + 5;")
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

	if cliOptions.printTokens {
		for _, token := range tokens {
			fmt.Fprintln(os.Stdout, driver.FormatToken(token))
		}
	}

	needParsed := cliOptions.printAST || cliOptions.printMermaid || cliOptions.printSemantic || cliOptions.runProgram
	if !needParsed {
		return
	}

	mode := options.Mode{CompatLoginov: cliOptions.compatLoginov}
	parse := parser.NewParserWithOptions(tokens, mode)
	statements, err := parse.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cliOptions.printAST {
		printer := ast.NewAstPrinter()
		fmt.Print(printer.Print(statements))
	}

	if cliOptions.printMermaid {
		generator := ast.NewMermaidGenerator()
		fmt.Print(generator.Generate(statements))
	}

	analyzer := semantic.NewSemanticAnalyzerWithOptions(mode)
	analyzer.Analyze(statements)
	for _, diagnostic := range analyzer.Diagnostics() {
		fmt.Fprintln(os.Stderr, diagnostic)
	}
	if analyzer.HasErrors() {
		os.Exit(1)
	}

	if !cliOptions.runProgram {
		return
	}

	run := executor.NewExecutorWithOptions(os.Stdout, mode)
	if err := run.Execute(statements); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type cliOptions struct {
	filePath       string
	printTokens    bool
	printAST       bool
	printMermaid   bool
	printSemantic  bool
	runProgram     bool
	compatLoginov  bool
	explicitAction bool
}

func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	for _, arg := range args {
		switch arg {
		case "--tokens":
			opts.printTokens = true
			opts.explicitAction = true
		case "--ast":
			opts.printAST = true
			opts.explicitAction = true
		case "--mermaid":
			opts.printMermaid = true
			opts.explicitAction = true
		case "--semantic":
			opts.printSemantic = true
			opts.explicitAction = true
		case "--run":
			opts.runProgram = true
			opts.explicitAction = true
		case "--compat-loginov":
			opts.compatLoginov = true
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return cliOptions{}, fmt.Errorf("unknown option: %s", arg)
			}
			if opts.filePath != "" {
				return cliOptions{}, fmt.Errorf("multiple input files are not supported")
			}
			opts.filePath = arg
		}
	}

	if !opts.explicitAction {
		opts.runProgram = true
	}

	return opts, nil
}
