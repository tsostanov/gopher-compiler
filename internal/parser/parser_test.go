package parser

import (
	"comp/internal/ast"
	"comp/internal/lexer"
	tok "comp/internal/token"
	"testing"
)

func TestParserAllowsUntypedFunDeclaration(t *testing.T) {
	statements, err := parseSource(t, `fun add(a, b) { return a + b; }`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(statements))
	}

	function, ok := statements[0].(ast.FuncStmt)
	if !ok {
		t.Fatalf("expected function statement, got %T", statements[0])
	}
	if len(function.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(function.Parameters))
	}
	if function.Parameters[0].Type.Kind != ast.TypeUnknown || function.Parameters[1].Type.Kind != ast.TypeUnknown {
		t.Fatalf("expected untyped parameters, got %#v", function.Parameters)
	}
	if function.ReturnType.Kind != ast.TypeUnknown {
		t.Fatalf("expected unknown return type, got %s", function.ReturnType.Kind.String())
	}
}

func TestParserAllowsReturnWithoutValue(t *testing.T) {
	statements, err := parseSource(t, `fun noop() { return; }`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	function := statements[0].(ast.FuncStmt)
	returnStmt, ok := function.Body.Statements[0].(ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %T", function.Body.Statements[0])
	}
	if returnStmt.Value != nil {
		t.Fatalf("expected nil return value, got %T", returnStmt.Value)
	}
}

func TestParserRecoveryKeepsFollowingStatements(t *testing.T) {
	tokens := tokenizeSource(t, `var x = ; print 1;`)
	parse := NewParser(tokens)

	statements, errors := parse.ParseWithRecovery()
	if len(errors) == 0 {
		t.Fatalf("expected parse errors")
	}
	if len(statements) != 1 {
		t.Fatalf("expected one recovered statement, got %d", len(statements))
	}
	if _, ok := statements[0].(ast.PrintStmt); !ok {
		t.Fatalf("expected recovered print statement, got %T", statements[0])
	}
}

func parseSource(t *testing.T, source string) ([]ast.Stmt, error) {
	t.Helper()

	tokens := tokenizeSource(t, source)
	parse := NewParser(tokens)
	return parse.Parse()
}

func tokenizeSource(t *testing.T, source string) []tok.Token {
	t.Helper()

	lex := lexer.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}
	return tokens
}
