package optimizer

import (
	"comp/internal/ast"
	tok "comp/internal/token"
	"strconv"
)

type Optimizer struct{}

func NewOptimizer() *Optimizer {
	return &Optimizer{}
}

func (o *Optimizer) OptimizeStatements(statements []ast.Stmt) []ast.Stmt {
	optimized := make([]ast.Stmt, 0, len(statements))
	for _, statement := range statements {
		optimized = append(optimized, o.OptimizeStatement(statement))
	}
	return optimized
}

func (o *Optimizer) OptimizeStatement(stmt ast.Stmt) ast.Stmt {
	switch s := stmt.(type) {
	case ast.VarStmt:
		var declaredType *ast.TypeAnnotation
		if s.DeclaredType != nil {
			copied := *s.DeclaredType
			declaredType = &copied
		}

		var initializer ast.Expr
		if s.Initializer != nil {
			initializer = o.OptimizeExpression(s.Initializer)
		}

		return ast.VarStmt{
			Name:         s.Name,
			DeclaredType: declaredType,
			Initializer:  initializer,
		}
	case ast.PrintStmt:
		return ast.PrintStmt{Expression: o.OptimizeExpression(s.Expression)}
	case ast.ExprStmt:
		return ast.ExprStmt{Expression: o.OptimizeExpression(s.Expression)}
	case ast.BlockStmt:
		return ast.BlockStmt{Statements: o.OptimizeStatements(s.Statements)}
	case ast.IfStmt:
		var elseBranch ast.Stmt
		if s.ElseBranch != nil {
			elseBranch = o.OptimizeStatement(s.ElseBranch)
		}
		return ast.IfStmt{
			Keyword:    s.Keyword,
			Condition:  o.OptimizeExpression(s.Condition),
			ThenBranch: o.OptimizeStatement(s.ThenBranch),
			ElseBranch: elseBranch,
		}
	case ast.WhileStmt:
		return ast.WhileStmt{
			Keyword:   s.Keyword,
			Condition: o.OptimizeExpression(s.Condition),
			Body:      o.OptimizeStatement(s.Body),
		}
	case ast.FuncStmt:
		parameters := append([]ast.Parameter(nil), s.Parameters...)
		body := o.OptimizeStatement(s.Body).(ast.BlockStmt)
		return ast.FuncStmt{
			Name:       s.Name,
			Parameters: parameters,
			ReturnType: s.ReturnType,
			Body:       body,
		}
	case ast.ReturnStmt:
		var value ast.Expr
		if s.Value != nil {
			value = o.OptimizeExpression(s.Value)
		}
		return ast.ReturnStmt{Keyword: s.Keyword, Value: value}
	default:
		return stmt
	}
}

func (o *Optimizer) OptimizeExpression(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case ast.LiteralExpr:
		return e
	case ast.VariableExpr:
		return e
	case ast.GroupingExpr:
		inner := o.OptimizeExpression(e.Expression)
		if _, ok := inner.(ast.LiteralExpr); ok {
			return inner
		}
		return ast.GroupingExpr{Expression: inner}
	case ast.UnaryExpr:
		right := o.OptimizeExpression(e.Right)
		optimized := ast.UnaryExpr{Operator: e.Operator, Right: right}
		if literal, ok := right.(ast.LiteralExpr); ok {
			if folded, ok := foldUnary(e.Operator, literal); ok {
				return folded
			}
		}
		return optimized
	case ast.BinaryExpr:
		left := o.OptimizeExpression(e.Left)
		right := o.OptimizeExpression(e.Right)
		optimized := ast.BinaryExpr{Left: left, Operator: e.Operator, Right: right}

		leftLiteral, leftOK := left.(ast.LiteralExpr)
		rightLiteral, rightOK := right.(ast.LiteralExpr)
		if leftOK && rightOK {
			if folded, ok := foldBinary(e.Operator, leftLiteral, rightLiteral); ok {
				return folded
			}
		}
		return optimized
	case ast.AssignExpr:
		return ast.AssignExpr{Name: e.Name, Value: o.OptimizeExpression(e.Value)}
	case ast.CallExpr:
		arguments := make([]ast.Expr, 0, len(e.Arguments))
		for _, argument := range e.Arguments {
			arguments = append(arguments, o.OptimizeExpression(argument))
		}
		return ast.CallExpr{
			Callee:    o.OptimizeExpression(e.Callee),
			Paren:     e.Paren,
			Arguments: arguments,
		}
	default:
		return expr
	}
}

func foldUnary(operator tok.Token, right ast.LiteralExpr) (ast.Expr, bool) {
	switch operator.Type {
	case tok.TokenMinus:
		if right.Token.Type != tok.TokenNumber {
			return nil, false
		}
		value, err := strconv.Atoi(right.Token.Value)
		if err != nil {
			return nil, false
		}
		return literalNumber(operator, -value), true
	case tok.TokenExcl:
		switch right.Token.Type {
		case tok.TokenTrue:
			return literalBool(operator, false), true
		case tok.TokenFalse:
			return literalBool(operator, true), true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}

func foldBinary(operator tok.Token, left, right ast.LiteralExpr) (ast.Expr, bool) {
	switch operator.Type {
	case tok.TokenPlus:
		if left.Token.Type == tok.TokenNumber && right.Token.Type == tok.TokenNumber {
			leftValue, err := strconv.Atoi(left.Token.Value)
			if err != nil {
				return nil, false
			}
			rightValue, err := strconv.Atoi(right.Token.Value)
			if err != nil {
				return nil, false
			}
			return literalNumber(operator, leftValue+rightValue), true
		}
		if left.Token.Type == tok.TokenString && right.Token.Type == tok.TokenString {
			return literalString(operator, left.Token.Value+right.Token.Value), true
		}
	case tok.TokenMinus:
		return foldIntegerBinary(operator, left, right, func(a, b int) int { return a - b })
	case tok.TokenStar:
		return foldIntegerBinary(operator, left, right, func(a, b int) int { return a * b })
	case tok.TokenSlash:
		leftValue, rightValue, ok := parseIntegerLiterals(left, right)
		if !ok || rightValue == 0 {
			return nil, false
		}
		return literalNumber(operator, leftValue/rightValue), true
	}

	return nil, false
}

func foldIntegerBinary(operator tok.Token, left, right ast.LiteralExpr, operation func(int, int) int) (ast.Expr, bool) {
	leftValue, rightValue, ok := parseIntegerLiterals(left, right)
	if !ok {
		return nil, false
	}
	return literalNumber(operator, operation(leftValue, rightValue)), true
}

func parseIntegerLiterals(left, right ast.LiteralExpr) (int, int, bool) {
	if left.Token.Type != tok.TokenNumber || right.Token.Type != tok.TokenNumber {
		return 0, 0, false
	}

	leftValue, err := strconv.Atoi(left.Token.Value)
	if err != nil {
		return 0, 0, false
	}
	rightValue, err := strconv.Atoi(right.Token.Value)
	if err != nil {
		return 0, 0, false
	}
	return leftValue, rightValue, true
}

func literalNumber(template tok.Token, value int) ast.LiteralExpr {
	return ast.LiteralExpr{
		Token: tok.Token{
			Type:     tok.TokenNumber,
			Value:    strconv.Itoa(value),
			Position: template.Position,
			Line:     template.Line,
			Column:   template.Column,
		},
	}
}

func literalString(template tok.Token, value string) ast.LiteralExpr {
	return ast.LiteralExpr{
		Token: tok.Token{
			Type:     tok.TokenString,
			Value:    value,
			Position: template.Position,
			Line:     template.Line,
			Column:   template.Column,
		},
	}
}

func literalBool(template tok.Token, value bool) ast.LiteralExpr {
	tokenType := tok.TokenFalse
	tokenValue := "false"
	if value {
		tokenType = tok.TokenTrue
		tokenValue = "true"
	}

	return ast.LiteralExpr{
		Token: tok.Token{
			Type:     tokenType,
			Value:    tokenValue,
			Position: template.Position,
			Line:     template.Line,
			Column:   template.Column,
		},
	}
}
