package executor

import (
	"comp/internal/lexer"
	"comp/internal/options"
	"comp/internal/parser"
	"comp/internal/semantic"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutorRunsMainExampleProgram(t *testing.T) {
	source := readExampleFile(t, "program.txt")

	output, err := executeSource(t, source)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	expected := "large\n57\n0\n1\n2\n"
	if output != expected {
		t.Fatalf("expected output %q, got %q", expected, output)
	}
}

func TestExecutorRunsLoginovCompatibilityExamples(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{name: "loginov_style/basic.txt", expected: "10\n"},
		{name: "loginov_style/functions.txt", expected: "15\n"},
		{name: "loginov_style/if_while.txt", expected: "0\nmiddle\n2\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := readExampleFile(t, tc.name)

			output, err := executeSourceWithOptions(t, source, options.Mode{CompatLoginov: true})
			if err != nil {
				t.Fatalf("execute failed: %v", err)
			}
			if output != tc.expected {
				t.Fatalf("expected output %q, got %q", tc.expected, output)
			}
		})
	}
}

func readExampleFile(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "examples", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example %s failed: %v", name, err)
	}
	return string(data)
}

func executeSource(t *testing.T, source string) (string, error) {
	t.Helper()

	return executeSourceWithOptions(t, source, options.Mode{})
}

func executeSourceWithOptions(t *testing.T, source string, mode options.Mode) (string, error) {
	t.Helper()

	lex := lexer.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	parse := parser.NewParserWithOptions(tokens, mode)
	statements, err := parse.Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := semantic.NewSemanticAnalyzerWithOptions(mode)
	diagnostics := analyzer.Analyze(statements)
	if analyzer.HasErrors() {
		t.Fatalf("unexpected semantic errors: %#v", diagnostics)
	}

	var output strings.Builder
	executor := NewExecutorWithOptions(&output, mode)
	err = executor.Execute(statements)
	return output.String(), err
}
