package main

import (
	"comp/internal/ast"
	"comp/internal/driver"
	"comp/internal/executor"
	"comp/internal/lexer"
	"comp/internal/optimizer"
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

	needParsed := cliOptions.printAST || cliOptions.printMermaid || cliOptions.printSemantic || cliOptions.runProgram || cliOptions.printOptimizedAST || cliOptions.printASTBeforeAfter || cliOptions.verifyOptimization
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
		printAST(os.Stdout, statements)
	}

	if cliOptions.printMermaid {
		generator := ast.NewMermaidGenerator()
		fmt.Print(generator.Generate(statements))
	}

	analyzer := semantic.NewSemanticAnalyzerWithOptions(mode)
	analyzer.Analyze(statements)
	printDiagnostics(os.Stderr, analyzer.Diagnostics())
	if analyzer.HasErrors() {
		os.Exit(1)
	}

	opt := optimizer.NewOptimizer()
	optimizedStatements := opt.OptimizeStatements(statements)

	optimizedAnalyzer := semantic.NewSemanticAnalyzerWithOptions(mode)
	optimizedAnalyzer.Analyze(optimizedStatements)
	printDiagnostics(os.Stderr, optimizedAnalyzer.Diagnostics())
	if optimizedAnalyzer.HasErrors() {
		os.Exit(1)
	}

	if cliOptions.printASTBeforeAfter {
		fmt.Fprintln(os.Stdout, "=== AST before optimization ===")
		printAST(os.Stdout, statements)
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "=== AST after optimization ===")
		printAST(os.Stdout, optimizedStatements)
	}

	if cliOptions.printOptimizedAST {
		printAST(os.Stdout, optimizedStatements)
	}

	if cliOptions.verifyOptimization {
		verifier := optimizer.NewOptimizationVerifierWithOptions(mode)
		result := verifier.Verify(statements, optimizedStatements)
		fmt.Fprintln(os.Stdout, result.Message)
		if !result.Ok {
			if result.OriginalOutput != "" || result.OptimizedOutput != "" {
				fmt.Fprintln(os.Stdout, "Original output:")
				fmt.Fprint(os.Stdout, result.OriginalOutput)
				fmt.Fprintln(os.Stdout, "Optimized output:")
				fmt.Fprint(os.Stdout, result.OptimizedOutput)
			}
			if result.OriginalError != nil || result.OptimizedError != nil {
				fmt.Fprintf(os.Stdout, "Original error: %v\n", result.OriginalError)
				fmt.Fprintf(os.Stdout, "Optimized error: %v\n", result.OptimizedError)
			}
			os.Exit(1)
		}
	}

	if !cliOptions.runProgram {
		return
	}

	run := executor.NewExecutorWithOptions(os.Stdout, mode)
	if err := run.Execute(optimizedStatements); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type cliOptions struct {
	filePath            string
	printTokens         bool
	printAST            bool
	printMermaid        bool
	printSemantic       bool
	printOptimizedAST   bool
	printASTBeforeAfter bool
	verifyOptimization  bool
	runProgram          bool
	compatLoginov       bool
	explicitAction      bool
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
		case "--optimized-ast":
			opts.printOptimizedAST = true
			opts.explicitAction = true
		case "--ast-before-after":
			opts.printASTBeforeAfter = true
			opts.explicitAction = true
		case "--verify-optimization":
			opts.verifyOptimization = true
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

func printAST(output *os.File, statements []ast.Stmt) {
	printer := ast.NewAstPrinter()
	fmt.Fprint(output, printer.Print(statements))
}

func printDiagnostics(output *os.File, diagnostics []semantic.SemanticDiagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(output, diagnostic)
	}
}
