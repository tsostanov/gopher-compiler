package optimizer

import (
	"comp/internal/ast"
	"comp/internal/lexer"
	"comp/internal/parser"
	"comp/internal/semantic"
	"strings"
	"testing"
)

func TestOptimizerFoldsArithmeticConstants(t *testing.T) {
	statements := parseAndValidate(t, `print 2 + 3 * 4;`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	printStmt := optimized[0].(ast.PrintStmt)
	literal, ok := printStmt.Expression.(ast.LiteralExpr)
	if !ok {
		t.Fatalf("expected literal after optimization, got %T", printStmt.Expression)
	}
	if literal.Token.Value != "14" {
		t.Fatalf("expected folded value 14, got %q", literal.Token.Value)
	}
}

func TestOptimizerFoldsStringConcatenation(t *testing.T) {
	statements := parseAndValidate(t, `print "hello, " + "world";`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	printStmt := optimized[0].(ast.PrintStmt)
	literal, ok := printStmt.Expression.(ast.LiteralExpr)
	if !ok {
		t.Fatalf("expected literal after optimization, got %T", printStmt.Expression)
	}
	if literal.Token.Value != "hello, world" {
		t.Fatalf("expected folded string %q, got %q", "hello, world", literal.Token.Value)
	}
}

func TestOptimizerPartiallyOptimizesMixedExpression(t *testing.T) {
	statements := parseAndValidate(t, `
var x = 10;
print x + (2 + 3);
`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	printStmt := optimized[1].(ast.PrintStmt)
	binary, ok := printStmt.Expression.(ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected binary expression, got %T", printStmt.Expression)
	}

	if _, ok := binary.Left.(ast.VariableExpr); !ok {
		t.Fatalf("expected left side to stay variable, got %T", binary.Left)
	}
	right, ok := binary.Right.(ast.LiteralExpr)
	if !ok {
		t.Fatalf("expected right side to fold to literal, got %T", binary.Right)
	}
	if right.Token.Value != "5" {
		t.Fatalf("expected folded value 5, got %q", right.Token.Value)
	}
}

func TestOptimizerDoesNotFoldDivisionByZero(t *testing.T) {
	statements := parseAndValidate(t, `print 1 / 0;`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	printStmt := optimized[0].(ast.PrintStmt)
	binary, ok := printStmt.Expression.(ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected division to stay binary expression, got %T", printStmt.Expression)
	}
	if binary.Operator.Value != "/" {
		t.Fatalf("expected division operator, got %q", binary.Operator.Value)
	}

	result := NewOptimizationVerifier().Verify(statements, optimized)
	if !result.Ok {
		t.Fatalf("expected verifier success, got %s", result.Message)
	}
	if result.OriginalError == nil || result.OptimizedError == nil {
		t.Fatalf("expected both executions to fail with division by zero")
	}
}

func TestOptimizerDoesNotFoldFunctionCallsAway(t *testing.T) {
	statements := parseAndValidate(t, `
func get(): int {
	print "called";
	return 2;
}

print get() + 3;
`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	printStmt := optimized[1].(ast.PrintStmt)
	binary, ok := printStmt.Expression.(ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected binary expression, got %T", printStmt.Expression)
	}
	if _, ok := binary.Left.(ast.CallExpr); !ok {
		t.Fatalf("expected function call to remain in AST, got %T", binary.Left)
	}

	result := NewOptimizationVerifier().Verify(statements, optimized)
	if !result.Ok {
		t.Fatalf("expected verifier success, got %s", result.Message)
	}
	if !strings.Contains(result.OriginalOutput, "called\n5\n") {
		t.Fatalf("expected original output to include side effect and result, got %q", result.OriginalOutput)
	}
}

func TestOptimizerFoldsUnaryConstants(t *testing.T) {
	statements := parseAndValidate(t, `
print -(2 + 3);
print !false;
`)

	optimized := NewOptimizer().OptimizeStatements(statements)
	first := optimized[0].(ast.PrintStmt).Expression.(ast.LiteralExpr)
	second := optimized[1].(ast.PrintStmt).Expression.(ast.LiteralExpr)

	if first.Token.Value != "-5" {
		t.Fatalf("expected -5, got %q", first.Token.Value)
	}
	if second.Token.Value != "true" {
		t.Fatalf("expected true, got %q", second.Token.Value)
	}
}

func parseAndValidate(t *testing.T, source string) []ast.Stmt {
	t.Helper()

	lex := lexer.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	parse := parser.NewParser(tokens)
	statements, err := parse.Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := semantic.NewSemanticAnalyzer()
	diagnostics := analyzer.Analyze(statements)
	if analyzer.HasErrors() {
		t.Fatalf("unexpected semantic errors: %#v", diagnostics)
	}

	return statements
}
