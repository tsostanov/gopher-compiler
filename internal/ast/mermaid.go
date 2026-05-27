package ast

import (
	tok "comp/internal/token"
	"fmt"
	"strings"
)

type MermaidGenerator struct {
	counter int
	lines   []string
}

func NewMermaidGenerator() *MermaidGenerator {
	return &MermaidGenerator{}
}

func (g *MermaidGenerator) Generate(stmts []Stmt) string {
	g.counter = 0
	g.lines = []string{"graph TD"}

	root := g.addNode("Root (Program)")
	for _, stmt := range stmts {
		child := g.visitStatement(stmt)
		g.addEdge(root, child, "")
	}

	return strings.Join(g.lines, "\n") + "\n"
}

func (g *MermaidGenerator) visitStatement(stmt Stmt) string {
	switch s := stmt.(type) {
	case VarStmt:
		node := g.addNode("Var: " + s.Name.Value)
		if s.DeclaredType != nil {
			typeNode := g.addNode("Type: " + s.DeclaredType.Kind.String())
			g.addEdge(node, typeNode, "type")
		}
		if s.Initializer != nil {
			g.addEdge(node, g.visitExpression(s.Initializer), "init")
		}
		return node
	case PrintStmt:
		node := g.addNode("Print")
		g.addEdge(node, g.visitExpression(s.Expression), "value")
		return node
	case ExprStmt:
		node := g.addNode("ExprStmt")
		g.addEdge(node, g.visitExpression(s.Expression), "expr")
		return node
	case BlockStmt:
		node := g.addNode("Block")
		for _, nested := range s.Statements {
			g.addEdge(node, g.visitStatement(nested), "")
		}
		return node
	case IfStmt:
		node := g.addNode("If")
		g.addEdge(node, g.visitExpression(s.Condition), "cond")
		g.addEdge(node, g.visitStatement(s.ThenBranch), "then")
		if s.ElseBranch != nil {
			g.addEdge(node, g.visitStatement(s.ElseBranch), "else")
		}
		return node
	case WhileStmt:
		node := g.addNode("While")
		g.addEdge(node, g.visitExpression(s.Condition), "cond")
		g.addEdge(node, g.visitStatement(s.Body), "body")
		return node
	case FuncStmt:
		node := g.addNode("Func: " + s.Name.Value)
		for _, parameter := range s.Parameters {
			paramNode := g.addNode(parameterLabel(parameter))
			g.addEdge(node, paramNode, "param")
		}
		if s.ReturnType.Kind != TypeUnknown {
			returnNode := g.addNode("ReturnType: " + s.ReturnType.Kind.String())
			g.addEdge(node, returnNode, "returns")
		}
		g.addEdge(node, g.visitStatement(s.Body), "body")
		return node
	case ReturnStmt:
		node := g.addNode("Return")
		if s.Value != nil {
			g.addEdge(node, g.visitExpression(s.Value), "value")
		}
		return node
	default:
		return g.addNode(fmt.Sprintf("%T", stmt))
	}
}

func (g *MermaidGenerator) visitExpression(expr Expr) string {
	switch e := expr.(type) {
	case BinaryExpr:
		node := g.addNode("Binary: " + e.Operator.Value)
		g.addEdge(node, g.visitExpression(e.Left), "left")
		g.addEdge(node, g.visitExpression(e.Right), "right")
		return node
	case UnaryExpr:
		node := g.addNode("Unary: " + e.Operator.Value)
		g.addEdge(node, g.visitExpression(e.Right), "value")
		return node
	case AssignExpr:
		node := g.addNode("Assign: " + e.Name.Value)
		g.addEdge(node, g.visitExpression(e.Value), "value")
		return node
	case ArrayExpr:
		node := g.addNode("Array")
		for _, element := range e.Elements {
			g.addEdge(node, g.visitExpression(element), "item")
		}
		return node
	case IndexExpr:
		node := g.addNode("Index")
		g.addEdge(node, g.visitExpression(e.Target), "target")
		g.addEdge(node, g.visitExpression(e.Index), "index")
		return node
	case IndexAssignExpr:
		node := g.addNode("IndexAssign")
		g.addEdge(node, g.visitExpression(e.Target), "target")
		g.addEdge(node, g.visitExpression(e.Index), "index")
		g.addEdge(node, g.visitExpression(e.Value), "value")
		return node
	case CallExpr:
		node := g.addNode("Call")
		g.addEdge(node, g.visitExpression(e.Callee), "callee")
		for _, argument := range e.Arguments {
			g.addEdge(node, g.visitExpression(argument), "arg")
		}
		return node
	case LiteralExpr:
		return g.addNode(literalLabel(e.Token))
	case VariableExpr:
		return g.addNode("VarRef: " + e.Name.Value)
	case GroupingExpr:
		node := g.addNode("Grouping")
		g.addEdge(node, g.visitExpression(e.Expression), "expr")
		return node
	default:
		return g.addNode(fmt.Sprintf("%T", expr))
	}
}

func (g *MermaidGenerator) addNode(label string) string {
	id := fmt.Sprintf("node%d", g.counter)
	g.counter++
	g.lines = append(g.lines, fmt.Sprintf("    %s[%q]", id, label))
	return id
}

func (g *MermaidGenerator) addEdge(from, to, label string) {
	if label == "" {
		g.lines = append(g.lines, fmt.Sprintf("    %s --> %s", from, to))
		return
	}
	g.lines = append(g.lines, fmt.Sprintf("    %s -- %q --> %s", from, label, to))
}

func parameterLabel(parameter Parameter) string {
	if parameter.Type.Kind == TypeUnknown {
		return "Param: " + parameter.Name.Value
	}
	return "Param: " + parameter.Name.Value + " : " + parameter.Type.Kind.String()
}

func literalLabel(token tok.Token) string {
	switch token.Type {
	case tok.TokenString:
		return fmt.Sprintf("Literal: %q", token.Value)
	default:
		return "Literal: " + token.Value
	}
}
