package ast

import tok "comp/internal/token"

type Expr interface {
	exprNode()
}

type Stmt interface {
	stmtNode()
}

type LiteralExpr struct {
	Token tok.Token
}

func (LiteralExpr) exprNode() {}

type VariableExpr struct {
	Name tok.Token
}

func (VariableExpr) exprNode() {}

type UnaryExpr struct {
	Operator tok.Token
	Right    Expr
}

func (UnaryExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr
	Operator tok.Token
	Right    Expr
}

func (BinaryExpr) exprNode() {}

type CallExpr struct {
	Callee    Expr
	Paren     tok.Token
	Arguments []Expr
}

func (CallExpr) exprNode() {}

type AssignExpr struct {
	Name  tok.Token
	Value Expr
}

func (AssignExpr) exprNode() {}

type GroupingExpr struct {
	Expression Expr
}

func (GroupingExpr) exprNode() {}

type Parameter struct {
	Name tok.Token
	Type TypeAnnotation
}

type VarStmt struct {
	Name         tok.Token
	DeclaredType *TypeAnnotation
	Initializer  Expr
}

func (VarStmt) stmtNode() {}

type PrintStmt struct {
	Expression Expr
}

func (PrintStmt) stmtNode() {}

type ExprStmt struct {
	Expression Expr
}

func (ExprStmt) stmtNode() {}

type BlockStmt struct {
	Statements []Stmt
}

func (BlockStmt) stmtNode() {}

type IfStmt struct {
	Keyword    tok.Token
	Condition  Expr
	ThenBranch Stmt
	ElseBranch Stmt
}

func (IfStmt) stmtNode() {}

type WhileStmt struct {
	Keyword   tok.Token
	Condition Expr
	Body      Stmt
}

func (WhileStmt) stmtNode() {}

type FuncStmt struct {
	Name       tok.Token
	Parameters []Parameter
	ReturnType TypeAnnotation
	Body       BlockStmt
}

func (FuncStmt) stmtNode() {}

type ReturnStmt struct {
	Keyword tok.Token
	Value   Expr
}

func (ReturnStmt) stmtNode() {}
