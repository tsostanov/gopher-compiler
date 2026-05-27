package parser

import (
	"comp/internal/ast"
	"comp/internal/options"
	tok "comp/internal/token"
	"fmt"
)

type ParseError struct {
	Message string
	Line    int
	Column  int
}

func (e ParseError) Error() string {
	return fmt.Sprintf("parse error at %d:%d: %s", e.Line, e.Column, e.Message)
}

type Parser struct {
	tokens   []tok.Token
	position int
	options  options.Mode
	errors   []ParseError
}

func NewParser(tokens []tok.Token) *Parser {
	return NewParserWithOptions(tokens, options.Mode{})
}

func NewParserWithOptions(tokens []tok.Token, mode options.Mode) *Parser {
	return &Parser{
		tokens:  tokens,
		options: mode,
	}
}

func (p *Parser) Parse() ([]ast.Stmt, error) {
	var statements []ast.Stmt
	for !p.isAtEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}
	return statements, nil
}

func (p *Parser) ParseWithRecovery() ([]ast.Stmt, []ParseError) {
	p.errors = nil

	var statements []ast.Stmt
	for !p.isAtEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			p.errors = append(p.errors, asParseError(err))
			p.synchronize()
			continue
		}
		statements = append(statements, stmt)
	}

	errors := make([]ParseError, len(p.errors))
	copy(errors, p.errors)
	return statements, errors
}

func (p *Parser) parseStatement() (ast.Stmt, error) {
	if p.match(tok.TokenVar) {
		return p.parseVarDeclaration()
	}
	if p.match(tok.TokenFunc) {
		return p.parseFunctionDeclaration(p.previous())
	}
	if p.match(tok.TokenReturn) {
		return p.parseReturnStatement()
	}
	if p.match(tok.TokenPrint) {
		return p.parsePrintStatement()
	}
	if p.match(tok.TokenIf) {
		return p.parseIfStatement()
	}
	if p.match(tok.TokenWhile) {
		return p.parseWhileStatement()
	}
	if p.match(tok.TokenLBrace) {
		return p.parseBlockStatement()
	}
	return p.parseExpressionStatement()
}

func (p *Parser) parseVarDeclaration() (ast.Stmt, error) {
	name, err := p.consume(tok.TokenID, "expected variable name")
	if err != nil {
		return nil, err
	}

	var declaredType *ast.TypeAnnotation
	if p.match(tok.TokenColon) {
		declaredType, err = p.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
	}

	var initializer ast.Expr
	if p.match(tok.TokenEq) {
		initializer, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}

	if _, err := p.consume(tok.TokenSemicolon, "expected ';' after variable declaration"); err != nil {
		return nil, err
	}

	return ast.VarStmt{Name: name, DeclaredType: declaredType, Initializer: initializer}, nil
}

func (p *Parser) parseFunctionDeclaration(keyword tok.Token) (ast.Stmt, error) {
	name, err := p.consume(tok.TokenID, "expected function name")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tok.TokenLParen, "expected '(' after function name"); err != nil {
		return nil, err
	}

	compatFunctionSyntax := keyword.Value == "fun" || p.options.CompatLoginov

	var parameters []ast.Parameter
	if !p.check(tok.TokenRParen) {
		for {
			paramName, err := p.consume(tok.TokenID, "expected parameter name")
			if err != nil {
				return nil, err
			}

			paramType := ast.TypeAnnotation{Kind: ast.TypeUnknown}
			if p.match(tok.TokenColon) {
				parsedType, err := p.parseTypeAnnotation()
				if err != nil {
					return nil, err
				}
				paramType = *parsedType
			} else if !compatFunctionSyntax {
				return nil, ParseError{
					Message: "expected ':' after parameter name",
					Line:    p.peek().Line,
					Column:  p.peek().Column,
				}
			}

			parameters = append(parameters, ast.Parameter{Name: paramName, Type: paramType})
			if !p.match(tok.TokenComma) {
				break
			}
		}
	}

	if _, err := p.consume(tok.TokenRParen, "expected ')' after parameter list"); err != nil {
		return nil, err
	}

	returnType := ast.TypeAnnotation{Kind: ast.TypeUnknown}
	if p.match(tok.TokenColon) {
		parsedType, err := p.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		returnType = *parsedType
	} else if !compatFunctionSyntax {
		return nil, ParseError{
			Message: "expected ':' before function return type",
			Line:    p.peek().Line,
			Column:  p.peek().Column,
		}
	}

	if _, err := p.consume(tok.TokenLBrace, "expected '{' before function body"); err != nil {
		return nil, err
	}
	body, err := p.parseBlockStatement()
	if err != nil {
		return nil, err
	}
	block, ok := body.(ast.BlockStmt)
	if !ok {
		return nil, ParseError{
			Message: "expected block statement in function body",
			Line:    name.Line,
			Column:  name.Column,
		}
	}

	return ast.FuncStmt{
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		Body:       block,
	}, nil
}

func (p *Parser) parsePrintStatement() (ast.Stmt, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tok.TokenSemicolon, "expected ';' after print statement"); err != nil {
		return nil, err
	}
	return ast.PrintStmt{Expression: expr}, nil
}

func (p *Parser) parseIfStatement() (ast.Stmt, error) {
	keyword := p.previous()
	if _, err := p.consume(tok.TokenLParen, "expected '(' after 'if'"); err != nil {
		return nil, err
	}
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tok.TokenRParen, "expected ')' after if condition"); err != nil {
		return nil, err
	}

	thenBranch, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	var elseBranch ast.Stmt
	if p.match(tok.TokenElse) {
		elseBranch, err = p.parseStatement()
		if err != nil {
			return nil, err
		}
	}

	return ast.IfStmt{Keyword: keyword, Condition: condition, ThenBranch: thenBranch, ElseBranch: elseBranch}, nil
}

func (p *Parser) parseWhileStatement() (ast.Stmt, error) {
	keyword := p.previous()
	if _, err := p.consume(tok.TokenLParen, "expected '(' after 'while'"); err != nil {
		return nil, err
	}
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tok.TokenRParen, "expected ')' after while condition"); err != nil {
		return nil, err
	}

	body, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	return ast.WhileStmt{Keyword: keyword, Condition: condition, Body: body}, nil
}

func (p *Parser) parseReturnStatement() (ast.Stmt, error) {
	keyword := p.previous()

	var value ast.Expr
	var err error
	if !p.check(tok.TokenSemicolon) {
		value, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}

	if _, err := p.consume(tok.TokenSemicolon, "expected ';' after return statement"); err != nil {
		return nil, err
	}
	return ast.ReturnStmt{Keyword: keyword, Value: value}, nil
}

func (p *Parser) parseBlockStatement() (ast.Stmt, error) {
	var statements []ast.Stmt
	for !p.check(tok.TokenRBrace) && !p.isAtEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}
	if _, err := p.consume(tok.TokenRBrace, "expected '}' after block"); err != nil {
		return nil, err
	}
	return ast.BlockStmt{Statements: statements}, nil
}

func (p *Parser) parseExpressionStatement() (ast.Stmt, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(tok.TokenSemicolon, "expected ';' after expression"); err != nil {
		return nil, err
	}
	return ast.ExprStmt{Expression: expr}, nil
}

func (p *Parser) parseExpression() (ast.Expr, error) {
	return p.parseAssignment()
}

func (p *Parser) parseAssignment() (ast.Expr, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.match(tok.TokenEq) {
		equals := p.previous()
		value, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}

		if varExpr, ok := expr.(ast.VariableExpr); ok {
			return ast.AssignExpr{Name: varExpr.Name, Value: value}, nil
		}
		if indexExpr, ok := expr.(ast.IndexExpr); ok {
			return ast.IndexAssignExpr{
				Target:  indexExpr.Target,
				Bracket: indexExpr.Bracket,
				Index:   indexExpr.Index,
				Value:   value,
			}, nil
		}

		return nil, ParseError{
			Message: "invalid assignment target",
			Line:    equals.Line,
			Column:  equals.Column,
		}
	}

	return expr, nil
}

func (p *Parser) parseOr() (ast.Expr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenOr) {
		operator := p.previous()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseAnd() (ast.Expr, error) {
	expr, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenAnd) {
		operator := p.previous()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseEquality() (ast.Expr, error) {
	expr, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenEqEq, tok.TokenNeq) {
		operator := p.previous()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseComparison() (ast.Expr, error) {
	expr, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenLt, tok.TokenLtEq, tok.TokenGt, tok.TokenGtEq) {
		operator := p.previous()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseTerm() (ast.Expr, error) {
	expr, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenPlus, tok.TokenMinus) {
		operator := p.previous()
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseFactor() (ast.Expr, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.match(tok.TokenStar, tok.TokenSlash) {
		operator := p.previous()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		expr = ast.BinaryExpr{Left: expr, Operator: operator, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.match(tok.TokenExcl, tok.TokenMinus) {
		operator := p.previous()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return ast.UnaryExpr{Operator: operator, Right: right}, nil
	}
	return p.parseCall()
}

func (p *Parser) parseCall() (ast.Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		if p.match(tok.TokenLParen) {
			paren := p.previous()
			var arguments []ast.Expr
			if !p.check(tok.TokenRParen) {
				for {
					argument, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					arguments = append(arguments, argument)
					if !p.match(tok.TokenComma) {
						break
					}
				}
			}

			if _, err := p.consume(tok.TokenRParen, "expected ')' after arguments"); err != nil {
				return nil, err
			}
			expr = ast.CallExpr{Callee: expr, Paren: paren, Arguments: arguments}
			continue
		}

		if p.match(tok.TokenLBracket) {
			bracket := p.previous()
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err := p.consume(tok.TokenRBracket, "expected ']' after index"); err != nil {
				return nil, err
			}
			expr = ast.IndexExpr{Target: expr, Bracket: bracket, Index: index}
			continue
		}

		break
	}

	return expr, nil
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	if p.match(tok.TokenNumber, tok.TokenString, tok.TokenTrue, tok.TokenFalse) {
		return ast.LiteralExpr{Token: p.previous()}, nil
	}
	if p.match(tok.TokenLBracket) {
		bracket := p.previous()
		var elements []ast.Expr
		if !p.check(tok.TokenRBracket) {
			for {
				element, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				elements = append(elements, element)
				if !p.match(tok.TokenComma) {
					break
				}
			}
		}
		if _, err := p.consume(tok.TokenRBracket, "expected ']' after array literal"); err != nil {
			return nil, err
		}
		return ast.ArrayExpr{Bracket: bracket, Elements: elements}, nil
	}
	if p.match(tok.TokenID) {
		return ast.VariableExpr{Name: p.previous()}, nil
	}
	if p.match(tok.TokenLParen) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(tok.TokenRParen, "expected ')' after expression"); err != nil {
			return nil, err
		}
		return ast.GroupingExpr{Expression: expr}, nil
	}

	token := p.peek()
	return nil, ParseError{
		Message: "expected expression",
		Line:    token.Line,
		Column:  token.Column,
	}
}

func (p *Parser) parseTypeAnnotation() (*ast.TypeAnnotation, error) {
	var valueType ast.ValueType
	var name tok.Token

	switch {
	case p.match(tok.TokenInt):
		name = p.previous()
		valueType = ast.TypeInt
	case p.match(tok.TokenBool):
		name = p.previous()
		valueType = ast.TypeBool
	case p.match(tok.TokenStringType):
		name = p.previous()
		valueType = ast.TypeString
	default:
		token := p.peek()
		return nil, ParseError{
			Message: "expected type name",
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	for p.match(tok.TokenLBracket) {
		if _, err := p.consume(tok.TokenRBracket, "expected ']' after '[' in array type"); err != nil {
			return nil, err
		}
		valueType = ast.ArrayOf(valueType)
	}

	return &ast.TypeAnnotation{Name: name, Kind: valueType}, nil
}

func (p *Parser) synchronize() {
	if p.isAtEnd() {
		return
	}

	p.advance()
	for !p.isAtEnd() {
		if p.previous().Type == tok.TokenSemicolon {
			return
		}

		switch p.peek().Type {
		case tok.TokenVar, tok.TokenFunc, tok.TokenPrint, tok.TokenIf, tok.TokenWhile, tok.TokenReturn, tok.TokenRBrace:
			return
		}

		p.advance()
	}
}

func asParseError(err error) ParseError {
	if parseErr, ok := err.(ParseError); ok {
		return parseErr
	}
	return ParseError{Message: err.Error()}
}

func (p *Parser) match(types ...tok.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) consume(t tok.TokenType, message string) (tok.Token, error) {
	if p.check(t) {
		return p.advance(), nil
	}
	token := p.peek()
	return tok.Token{}, ParseError{
		Message: message,
		Line:    token.Line,
		Column:  token.Column,
	}
}

func (p *Parser) check(t tok.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == t
}

func (p *Parser) advance() tok.Token {
	if !p.isAtEnd() {
		p.position++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == tok.TokenEOF
}

func (p *Parser) peek() tok.Token {
	if p.position >= len(p.tokens) {
		return tok.Token{Type: tok.TokenEOF}
	}
	return p.tokens[p.position]
}

func (p *Parser) previous() tok.Token {
	if p.position-1 < 0 || p.position-1 >= len(p.tokens) {
		return tok.Token{Type: tok.TokenEOF}
	}
	return p.tokens[p.position-1]
}
