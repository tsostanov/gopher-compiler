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

func TestCompatibilityModeProgramsBehaveInCompatibilityMode(t *testing.T) {
	testCases := []struct {
		name   string
		source string
	}{
		{
			name: "basic",
			source: `
var x;
x = 10;
print x;
`,
		},
		{
			name: "functions",
			source: `
fun add(a, b) {
	return a + b;
}
print add(7, 8);
`,
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
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics, err := analyzeExampleSourceWithOptions(tc.source, options.Mode{CompatLoginov: true})
			if err != nil {
				t.Fatalf("expected example %s to parse, got %v", tc.name, err)
			}
			if hasErrorDiagnostics(diagnostics) {
				t.Fatalf("expected no semantic errors for %s, got %#v", tc.name, diagnostics)
			}
		})
	}
}

func TestExampleErrorProgramsFailAsExpected(t *testing.T) {
	testCases := []struct {
		name          string
		source        string
		expectedError string
		parseError    bool
	}{
		{name: "bad_type_name", source: `var int: int = 1;`, expectedError: "expected variable name", parseError: true},
		{name: "unknown_type", source: `var x: number = 1;`, expectedError: "expected type name", parseError: true},
		{name: "if_condition", source: `if (1) { print 1; }`, expectedError: "if condition must have type bool"},
		{name: "string_compare", source: `print "a" < "b";`, expectedError: "comparison operators expect operands of type int"},
		{name: "types", source: `var flag: bool = true; flag = 1;`, expectedError: "cannot assign value of type int to variable flag of type bool"},
		{name: "undeclared", source: `print x;`, expectedError: "use of undeclared variable x"},
		{name: "uninitialized", source: `var x: int; print x;`, expectedError: "used before initialization"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics, err := analyzeExampleSource(tc.source)
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
