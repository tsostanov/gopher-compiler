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

	expected := "3\n10\n34\n"
	if output != expected {
		t.Fatalf("expected output %q, got %q", expected, output)
	}
}

func TestExecutorRunsCompatibilityPrograms(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name: "basic",
			source: `
var x;
x = 10;
print x;
`,
			expected: "10\n",
		},
		{
			name: "functions",
			source: `
fun add(a, b) {
	return a + b;
}
print add(7, 8);
`,
			expected: "15\n",
		},
		{
			name: "if_while",
			source: `
var i;
i = 0;
while (i < 3) {
	if (i == 1) {
		print "middle";
	} else {
		print i;
	}
	i = i + 1;
}
`,
			expected: "0\nmiddle\n2\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := executeSourceWithOptions(t, tc.source, options.Mode{CompatLoginov: true})
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
