package semantic

import (
	"comp/internal/lexer"
	"comp/internal/options"
	"comp/internal/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesProgramIsSemanticallyValid(t *testing.T) {
	source := readExampleFile(t, "program.txt")

	diagnostics, err := analyzeExampleSource(source)
	if err != nil {
		t.Fatalf("expected example program to parse, got %v", err)
	}
	if hasErrorDiagnostics(diagnostics) {
		t.Fatalf("expected no semantic errors, got %#v", diagnostics)
	}
}

func TestLoginovStyleExamplesBehaveInCompatibilityMode(t *testing.T) {
	testCases := []string{
		"loginov_style/basic.txt",
		"loginov_style/functions.txt",
		"loginov_style/if_while.txt",
		"loginov_style/mermaid.txt",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			source := readExampleFile(t, name)

			diagnostics, err := analyzeExampleSourceWithOptions(source, options.Mode{CompatLoginov: true})
			if err != nil {
				t.Fatalf("expected example %s to parse, got %v", name, err)
			}
			if hasErrorDiagnostics(diagnostics) {
				t.Fatalf("expected no semantic errors for %s, got %#v", name, diagnostics)
			}
		})
	}
}

func TestExampleErrorProgramsFailAsExpected(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError string
		parseError    bool
	}{
		{name: "program_bad_type_name.txt", expectedError: "expected variable name", parseError: true},
		{name: "program_unknown_type.txt", expectedError: "expected type name", parseError: true},
		{name: "program_error_if_condition.txt", expectedError: "if condition must have type bool"},
		{name: "program_error_string_compare.txt", expectedError: "comparison operators expect operands of type int"},
		{name: "program_error_types.txt", expectedError: "cannot assign value of type int to variable flag of type bool"},
		{name: "program_error_undeclared.txt", expectedError: "use of undeclared variable x"},
		{name: "program_error_uninitialized.txt", expectedError: "used before initialization"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := readExampleFile(t, tc.name)

			diagnostics, err := analyzeExampleSource(source)
			if tc.parseError {
				if err == nil {
					t.Fatalf("expected parse error containing %q", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected parse error containing %q, got %v", tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			assertHasDiagnostic(t, diagnostics, SeverityError, tc.expectedError)
		})
	}
}

func analyzeExampleSource(source string) ([]SemanticDiagnostic, error) {
	return analyzeExampleSourceWithOptions(source, options.Mode{})
}

func analyzeExampleSourceWithOptions(source string, mode options.Mode) ([]SemanticDiagnostic, error) {
	lex := lexer.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		return nil, err
	}

	parse := parser.NewParserWithOptions(tokens, mode)
	statements, err := parse.Parse()
	if err != nil {
		return nil, err
	}

	analyzer := NewSemanticAnalyzerWithOptions(mode)
	return analyzer.Analyze(statements), nil
}

func hasErrorDiagnostics(diagnostics []SemanticDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
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
