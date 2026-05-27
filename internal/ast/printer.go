package ast

import (
	tok "comp/internal/token"
	"fmt"
	"strings"
)

type AstPrinter struct{}

func NewAstPrinter() *AstPrinter {
	return &AstPrinter{}
}

func (p *AstPrinter) Print(stmts []Stmt) string {
	var b strings.Builder
	b.WriteString("Root (Program)\n")
	for i, stmt := range stmts {
		p.printNode(&b, stmt, "", i == len(stmts)-1)
	}
	return b.String()
}

func (p *AstPrinter) printNode(b *strings.Builder, node any, indent string, isLast bool) {
	if node == nil {
		return
	}

	marker := "|-- "
	if isLast {
		marker = "`-- "
	}
	b.WriteString(indent)
	b.WriteString(marker)

	childIndent := indent + "|   "
	if isLast {
		childIndent = indent + "    "
	}

	switch n := node.(type) {
	case VarStmt:
		if n.DeclaredType != nil {
			fmt.Fprintf(b, "VarStatement: %s : %s\n", n.Name.Value, n.DeclaredType.Kind.String())
		} else {
			fmt.Fprintf(b, "VarStatement: %s\n", n.Name.Value)
		}
		if n.Initializer != nil {
			p.printNode(b, n.Initializer, childIndent, true)
		}
	case PrintStmt:
		b.WriteString("PrintStatement\n")
		p.printNode(b, n.Expression, childIndent, true)
	case ExprStmt:
		b.WriteString("ExpressionStatement\n")
		p.printNode(b, n.Expression, childIndent, true)
	case BlockStmt:
		b.WriteString("BlockStatement\n")
		for i, stmt := range n.Statements {
			p.printNode(b, stmt, childIndent, i == len(n.Statements)-1)
		}
	case IfStmt:
		b.WriteString("IfStatement\n")
		p.printNode(b, n.Condition, childIndent, false)
		isThenLast := n.ElseBranch == nil
		p.printNode(b, n.ThenBranch, childIndent, isThenLast)
		if n.ElseBranch != nil {
			p.printNode(b, n.ElseBranch, childIndent, true)
		}
	case WhileStmt:
		b.WriteString("WhileStatement\n")
		p.printNode(b, n.Condition, childIndent, false)
		p.printNode(b, n.Body, childIndent, true)
	case FuncStmt:
		if n.ReturnType.Kind == TypeUnknown {
			fmt.Fprintf(b, "FunctionStatement: %s\n", n.Name.Value)
		} else {
			fmt.Fprintf(b, "FunctionStatement: %s -> %s\n", n.Name.Value, n.ReturnType.Kind.String())
		}
		for i, param := range n.Parameters {
			isLastParam := len(n.Body.Statements) == 0 && i == len(n.Parameters)-1
			p.printNode(b, param, childIndent, isLastParam)
		}
		for i, stmt := range n.Body.Statements {
			p.printNode(b, stmt, childIndent, i == len(n.Body.Statements)-1)
		}
	case ReturnStmt:
		b.WriteString("ReturnStatement\n")
		p.printNode(b, n.Value, childIndent, true)
	case BinaryExpr:
		fmt.Fprintf(b, "BinaryExpression: %s\n", n.Operator.Value)
		p.printNode(b, n.Left, childIndent, false)
		p.printNode(b, n.Right, childIndent, true)
	case CallExpr:
		b.WriteString("CallExpression\n")
		if len(n.Arguments) == 0 {
			p.printNode(b, n.Callee, childIndent, true)
			return
		}
		p.printNode(b, n.Callee, childIndent, false)
		for i, arg := range n.Arguments {
			p.printNode(b, arg, childIndent, i == len(n.Arguments)-1)
		}
	case AssignExpr:
		fmt.Fprintf(b, "AssignExpression: %s =\n", n.Name.Value)
		p.printNode(b, n.Value, childIndent, true)
	case ArrayExpr:
		b.WriteString("ArrayExpression\n")
		for i, element := range n.Elements {
			p.printNode(b, element, childIndent, i == len(n.Elements)-1)
		}
	case IndexExpr:
		b.WriteString("IndexExpression\n")
		p.printNode(b, n.Target, childIndent, false)
		p.printNode(b, n.Index, childIndent, true)
	case IndexAssignExpr:
		b.WriteString("IndexAssignExpression\n")
		p.printNode(b, n.Target, childIndent, false)
		p.printNode(b, n.Index, childIndent, false)
		p.printNode(b, n.Value, childIndent, true)
	case UnaryExpr:
		fmt.Fprintf(b, "UnaryExpression: %s\n", n.Operator.Value)
		p.printNode(b, n.Right, childIndent, true)
	case LiteralExpr:
		fmt.Fprintf(b, "Literal: %s\n", formatLiteral(n.Token))
	case VariableExpr:
		fmt.Fprintf(b, "Variable: %s\n", n.Name.Value)
	case GroupingExpr:
		b.WriteString("Grouping\n")
		p.printNode(b, n.Expression, childIndent, true)
	case Parameter:
		if n.Type.Kind == TypeUnknown {
			fmt.Fprintf(b, "Parameter: %s\n", n.Name.Value)
		} else {
			fmt.Fprintf(b, "Parameter: %s : %s\n", n.Name.Value, n.Type.Kind.String())
		}
	default:
		fmt.Fprintf(b, "Unknown Node: %T\n", node)
	}
}

func formatLiteral(token tok.Token) string {
	if token.Type == tok.TokenString {
		return fmt.Sprintf("%q", token.Value)
	}
	return token.Value
}
