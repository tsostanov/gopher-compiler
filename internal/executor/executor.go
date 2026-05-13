package executor

import (
	"comp/internal/ast"
	"comp/internal/options"
	tok "comp/internal/token"
	"fmt"
	"io"
	"strconv"
)

type RuntimeError struct {
	Message string
	Line    int
	Column  int
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: %s", e.Line, e.Column, e.Message)
}

type Value struct {
	Type ast.ValueType
	Data any
}

func (v Value) String() string {
	switch v.Type {
	case ast.TypeInt:
		return strconv.Itoa(v.Data.(int))
	case ast.TypeBool:
		if v.Data.(bool) {
			return "true"
		}
		return "false"
	case ast.TypeString:
		return v.Data.(string)
	default:
		return "<unknown>"
	}
}

type VariableSlot struct {
	Type        ast.ValueType
	Value       Value
	Initialized bool
}

type FunctionValue struct {
	Declaration ast.FuncStmt
	Closure     *Environment
}

type Environment struct {
	parent    *Environment
	variables map[string]*VariableSlot
	functions map[string]*FunctionValue
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		parent:    parent,
		variables: make(map[string]*VariableSlot),
		functions: make(map[string]*FunctionValue),
	}
}

func (e *Environment) DefineVariable(name string, variableType ast.ValueType, value Value, initialized bool) bool {
	if _, exists := e.variables[name]; exists {
		return false
	}
	if _, exists := e.functions[name]; exists {
		return false
	}

	e.variables[name] = &VariableSlot{
		Type:        variableType,
		Value:       value,
		Initialized: initialized,
	}
	return true
}

func (e *Environment) ResolveVariable(name string) *VariableSlot {
	for current := e; current != nil; current = current.parent {
		if variable, ok := current.variables[name]; ok {
			return variable
		}
	}
	return nil
}

func (e *Environment) DefineFunction(name string, function *FunctionValue) bool {
	if _, exists := e.functions[name]; exists {
		return false
	}
	if _, exists := e.variables[name]; exists {
		return false
	}

	e.functions[name] = function
	return true
}

func (e *Environment) ResolveFunction(name string) *FunctionValue {
	for current := e; current != nil; current = current.parent {
		if function, ok := current.functions[name]; ok {
			return function
		}
	}
	return nil
}

type statementResult struct {
	value    Value
	returned bool
}

type Executor struct {
	environment *Environment
	output      io.Writer
	options     options.Mode
}

func NewExecutor(output io.Writer) *Executor {
	return NewExecutorWithOptions(output, options.Mode{})
}

func NewExecutorWithOptions(output io.Writer, mode options.Mode) *Executor {
	return &Executor{
		environment: NewEnvironment(nil),
		output:      output,
		options:     mode,
	}
}

func (e *Executor) Execute(statements []ast.Stmt) error {
	_, returned, err := e.executeStatements(statements)
	if err != nil {
		return err
	}
	if returned {
		return RuntimeError{Message: "return statement is only allowed inside functions"}
	}
	return nil
}

func (e *Executor) executeStatements(statements []ast.Stmt) (Value, bool, error) {
	if err := e.predeclareFunctions(statements); err != nil {
		return Value{}, false, err
	}
	for _, statement := range statements {
		value, returned, err := e.executeStatement(statement)
		if err != nil {
			return Value{}, false, err
		}
		if returned {
			return value, true, nil
		}
	}
	return Value{}, false, nil
}

func (e *Executor) predeclareFunctions(statements []ast.Stmt) error {
	for _, statement := range statements {
		function, ok := statement.(ast.FuncStmt)
		if !ok {
			continue
		}
		if e.environment.functions[function.Name.Value] != nil {
			return runtimeError(function.Name, "name "+function.Name.Value+" is already declared in this scope")
		}
		if !e.environment.DefineFunction(function.Name.Value, &FunctionValue{
			Declaration: function,
			Closure:     e.environment,
		}) {
			return runtimeError(function.Name, "name "+function.Name.Value+" is already declared in this scope")
		}
	}
	return nil
}

func (e *Executor) executeStatement(statement ast.Stmt) (Value, bool, error) {
	switch s := statement.(type) {
	case ast.VarStmt:
		return Value{}, false, e.executeVarStatement(s)
	case ast.FuncStmt:
		return Value{}, false, nil
	case ast.ReturnStmt:
		if s.Value == nil {
			return Value{Type: ast.TypeUnknown, Data: nil}, true, nil
		}
		value, err := e.evaluateExpression(s.Value)
		if err != nil {
			return Value{}, false, err
		}
		return value, true, nil
	case ast.PrintStmt:
		value, err := e.evaluateExpression(s.Expression)
		if err != nil {
			return Value{}, false, err
		}
		_, err = fmt.Fprintln(e.output, value.String())
		return Value{}, false, err
	case ast.ExprStmt:
		_, err := e.evaluateExpression(s.Expression)
		return Value{}, false, err
	case ast.BlockStmt:
		return e.executeBlockStatement(s)
	case ast.IfStmt:
		condition, err := e.evaluateExpression(s.Condition)
		if err != nil {
			return Value{}, false, err
		}
		conditionValue, err := e.expectBool(condition, expressionToken(s.Condition), "if condition must be bool")
		if err != nil {
			return Value{}, false, err
		}
		if conditionValue {
			return e.executeStatement(s.ThenBranch)
		}
		if s.ElseBranch != nil {
			return e.executeStatement(s.ElseBranch)
		}
		return Value{}, false, nil
	case ast.WhileStmt:
		for {
			condition, err := e.evaluateExpression(s.Condition)
			if err != nil {
				return Value{}, false, err
			}
			conditionValue, err := e.expectBool(condition, expressionToken(s.Condition), "while condition must be bool")
			if err != nil {
				return Value{}, false, err
			}
			if !conditionValue {
				return Value{}, false, nil
			}
			value, returned, err := e.executeStatement(s.Body)
			if err != nil {
				return Value{}, false, err
			}
			if returned {
				return value, true, nil
			}
		}
	default:
		return Value{}, false, nil
	}
}

func (e *Executor) executeBlockStatement(statement ast.BlockStmt) (Value, bool, error) {
	previous := e.environment
	e.environment = NewEnvironment(previous)
	defer func() {
		e.environment = previous
	}()

	return e.executeStatements(statement.Statements)
}

func (e *Executor) executeVarStatement(statement ast.VarStmt) error {
	variableType := ast.TypeUnknown
	if statement.DeclaredType != nil {
		variableType = statement.DeclaredType.Kind
	}

	value := Value{Type: ast.TypeUnknown, Data: nil}
	initialized := false
	if statement.Initializer != nil {
		initializerValue, err := e.evaluateExpression(statement.Initializer)
		if err != nil {
			return err
		}
		if variableType == ast.TypeUnknown {
			variableType = initializerValue.Type
		}
		if variableType != ast.TypeUnknown && variableType != initializerValue.Type {
			return runtimeError(statement.Name, "cannot initialize variable "+statement.Name.Value+" of type "+variableType.String()+" with value of type "+initializerValue.Type.String())
		}
		value = initializerValue
		initialized = true
	} else {
		if variableType == ast.TypeUnknown && !e.options.CompatLoginov {
			return runtimeError(statement.Name, "variable "+statement.Name.Value+" requires an explicit type or initializer")
		}
		if variableType != ast.TypeUnknown {
			value = zeroValue(variableType)
		}
	}

	if !e.environment.DefineVariable(statement.Name.Value, variableType, value, initialized) {
		return runtimeError(statement.Name, "name "+statement.Name.Value+" is already declared in this scope")
	}
	return nil
}

func (e *Executor) evaluateExpression(expression ast.Expr) (Value, error) {
	switch expr := expression.(type) {
	case ast.LiteralExpr:
		return literalValue(expr.Token)
	case ast.VariableExpr:
		variable := e.environment.ResolveVariable(expr.Name.Value)
		if variable == nil {
			return Value{}, runtimeError(expr.Name, "use of undeclared variable "+expr.Name.Value)
		}
		if !variable.Initialized {
			return Value{}, runtimeError(expr.Name, "variable "+expr.Name.Value+" is used before initialization")
		}
		return variable.Value, nil
	case ast.AssignExpr:
		value, err := e.evaluateExpression(expr.Value)
		if err != nil {
			return Value{}, err
		}

		variable := e.environment.ResolveVariable(expr.Name.Value)
		if variable == nil {
			return Value{}, runtimeError(expr.Name, "assignment to undeclared variable "+expr.Name.Value)
		}
		if variable.Type != ast.TypeUnknown && variable.Type != value.Type {
			return Value{}, runtimeError(expr.Name, "cannot assign value of type "+value.Type.String()+" to variable "+expr.Name.Value+" of type "+variable.Type.String())
		}
		if variable.Type == ast.TypeUnknown {
			variable.Type = value.Type
		}

		variable.Value = value
		variable.Initialized = true
		return value, nil
	case ast.GroupingExpr:
		return e.evaluateExpression(expr.Expression)
	case ast.UnaryExpr:
		right, err := e.evaluateExpression(expr.Right)
		if err != nil {
			return Value{}, err
		}
		return e.evaluateUnary(expr.Operator, right)
	case ast.BinaryExpr:
		return e.evaluateBinary(expr)
	case ast.CallExpr:
		return e.evaluateCall(expr)
	default:
		return Value{}, RuntimeError{Message: "unsupported expression"}
	}
}

func (e *Executor) evaluateCall(expression ast.CallExpr) (Value, error) {
	callee, ok := expression.Callee.(ast.VariableExpr)
	if !ok {
		return Value{}, runtimeError(expression.Paren, "can only call named functions")
	}

	function := e.environment.ResolveFunction(callee.Name.Value)
	if function == nil {
		return Value{}, runtimeError(callee.Name, "call to undeclared function "+callee.Name.Value)
	}

	if len(expression.Arguments) != len(function.Declaration.Parameters) {
		return Value{}, runtimeError(callee.Name, fmt.Sprintf("function %s expects %d arguments, got %d", function.Declaration.Name.Value, len(function.Declaration.Parameters), len(expression.Arguments)))
	}

	arguments := make([]Value, 0, len(expression.Arguments))
	for _, argument := range expression.Arguments {
		value, err := e.evaluateExpression(argument)
		if err != nil {
			return Value{}, err
		}
		arguments = append(arguments, value)
	}

	return e.callFunction(function, arguments)
}

func (e *Executor) callFunction(function *FunctionValue, arguments []Value) (Value, error) {
	previous := e.environment
	callEnvironment := NewEnvironment(function.Closure)
	e.environment = callEnvironment
	defer func() {
		e.environment = previous
	}()

	for index, parameter := range function.Declaration.Parameters {
		argument := arguments[index]
		parameterType := parameter.Type.Kind
		if parameterType != ast.TypeUnknown && parameterType != argument.Type {
			return Value{}, runtimeError(parameter.Name, "cannot assign value of type "+argument.Type.String()+" to parameter "+parameter.Name.Value+" of type "+parameterType.String())
		}
		if parameterType == ast.TypeUnknown {
			parameterType = argument.Type
		}
		if !e.environment.DefineVariable(parameter.Name.Value, parameterType, argument, true) {
			return Value{}, runtimeError(parameter.Name, "name "+parameter.Name.Value+" is already declared in this scope")
		}
	}

	value, returned, err := e.executeStatements(function.Declaration.Body.Statements)
	if err != nil {
		return Value{}, err
	}
	if !returned {
		if function.Declaration.ReturnType.Kind == ast.TypeUnknown {
			return Value{Type: ast.TypeUnknown, Data: nil}, nil
		}
		return Value{}, runtimeError(function.Declaration.Name, "function "+function.Declaration.Name.Value+" did not return a value")
	}
	return value, nil
}

func (e *Executor) evaluateUnary(operator tok.Token, right Value) (Value, error) {
	switch operator.Type {
	case tok.TokenMinus:
		value, err := e.expectInt(right, operator, "operator - expects operand of type int")
		if err != nil {
			return Value{}, err
		}
		return Value{Type: ast.TypeInt, Data: -value}, nil
	case tok.TokenExcl:
		value, err := e.expectBool(right, operator, "operator ! expects operand of type bool")
		if err != nil {
			return Value{}, err
		}
		return Value{Type: ast.TypeBool, Data: !value}, nil
	default:
		return Value{}, runtimeError(operator, "unsupported unary operator "+operator.Value)
	}
}

func (e *Executor) evaluateBinary(expression ast.BinaryExpr) (Value, error) {
	switch expression.Operator.Type {
	case tok.TokenAnd:
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return Value{}, err
		}
		leftValue, err := e.expectBool(left, expression.Operator, "logical operators expect operands of type bool")
		if err != nil {
			return Value{}, err
		}
		if !leftValue {
			return Value{Type: ast.TypeBool, Data: false}, nil
		}
		right, err := e.evaluateExpression(expression.Right)
		if err != nil {
			return Value{}, err
		}
		rightValue, err := e.expectBool(right, expression.Operator, "logical operators expect operands of type bool")
		if err != nil {
			return Value{}, err
		}
		return Value{Type: ast.TypeBool, Data: rightValue}, nil
	case tok.TokenOr:
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return Value{}, err
		}
		leftValue, err := e.expectBool(left, expression.Operator, "logical operators expect operands of type bool")
		if err != nil {
			return Value{}, err
		}
		if leftValue {
			return Value{Type: ast.TypeBool, Data: true}, nil
		}
		right, err := e.evaluateExpression(expression.Right)
		if err != nil {
			return Value{}, err
		}
		rightValue, err := e.expectBool(right, expression.Operator, "logical operators expect operands of type bool")
		if err != nil {
			return Value{}, err
		}
		return Value{Type: ast.TypeBool, Data: rightValue}, nil
	}

	left, err := e.evaluateExpression(expression.Left)
	if err != nil {
		return Value{}, err
	}
	right, err := e.evaluateExpression(expression.Right)
	if err != nil {
		return Value{}, err
	}

	switch expression.Operator.Type {
	case tok.TokenPlus:
		if left.Type == ast.TypeInt && right.Type == ast.TypeInt {
			return Value{Type: ast.TypeInt, Data: left.Data.(int) + right.Data.(int)}, nil
		}
		if left.Type == ast.TypeString && right.Type == ast.TypeString {
			return Value{Type: ast.TypeString, Data: left.Data.(string) + right.Data.(string)}, nil
		}
		return Value{}, runtimeError(expression.Operator, "operator + expects operands of type int or string")
	case tok.TokenMinus:
		return e.evaluateIntBinary(expression.Operator, left, right, func(a, b int) (int, error) {
			return a - b, nil
		})
	case tok.TokenStar:
		return e.evaluateIntBinary(expression.Operator, left, right, func(a, b int) (int, error) {
			return a * b, nil
		})
	case tok.TokenSlash:
		return e.evaluateIntBinary(expression.Operator, left, right, func(a, b int) (int, error) {
			if b == 0 {
				return 0, runtimeError(expression.Operator, "division by zero")
			}
			return a / b, nil
		})
	case tok.TokenLt:
		return e.evaluateIntComparison(expression.Operator, left, right, func(a, b int) bool { return a < b })
	case tok.TokenLtEq:
		return e.evaluateIntComparison(expression.Operator, left, right, func(a, b int) bool { return a <= b })
	case tok.TokenGt:
		return e.evaluateIntComparison(expression.Operator, left, right, func(a, b int) bool { return a > b })
	case tok.TokenGtEq:
		return e.evaluateIntComparison(expression.Operator, left, right, func(a, b int) bool { return a >= b })
	case tok.TokenEqEq:
		return Value{Type: ast.TypeBool, Data: valuesEqual(left, right)}, nil
	case tok.TokenNeq:
		return Value{Type: ast.TypeBool, Data: !valuesEqual(left, right)}, nil
	default:
		return Value{}, runtimeError(expression.Operator, "unsupported binary operator "+expression.Operator.Value)
	}
}

func (e *Executor) evaluateIntBinary(operator tok.Token, left, right Value, operation func(int, int) (int, error)) (Value, error) {
	leftValue, err := e.expectInt(left, operator, "arithmetic operators expect operands of type int")
	if err != nil {
		return Value{}, err
	}
	rightValue, err := e.expectInt(right, operator, "arithmetic operators expect operands of type int")
	if err != nil {
		return Value{}, err
	}

	result, err := operation(leftValue, rightValue)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: ast.TypeInt, Data: result}, nil
}

func (e *Executor) evaluateIntComparison(operator tok.Token, left, right Value, compare func(int, int) bool) (Value, error) {
	leftValue, err := e.expectInt(left, operator, "comparison operators expect operands of type int")
	if err != nil {
		return Value{}, err
	}
	rightValue, err := e.expectInt(right, operator, "comparison operators expect operands of type int")
	if err != nil {
		return Value{}, err
	}

	return Value{Type: ast.TypeBool, Data: compare(leftValue, rightValue)}, nil
}

func (e *Executor) expectInt(value Value, token tok.Token, message string) (int, error) {
	if value.Type != ast.TypeInt {
		return 0, runtimeError(token, message)
	}
	return value.Data.(int), nil
}

func (e *Executor) expectBool(value Value, token tok.Token, message string) (bool, error) {
	if value.Type != ast.TypeBool {
		return false, runtimeError(token, message)
	}
	return value.Data.(bool), nil
}

func literalValue(token tok.Token) (Value, error) {
	switch token.Type {
	case tok.TokenNumber:
		value, err := strconv.Atoi(token.Value)
		if err != nil {
			return Value{}, runtimeError(token, "invalid integer literal "+token.Value)
		}
		return Value{Type: ast.TypeInt, Data: value}, nil
	case tok.TokenString:
		return Value{Type: ast.TypeString, Data: token.Value}, nil
	case tok.TokenTrue:
		return Value{Type: ast.TypeBool, Data: true}, nil
	case tok.TokenFalse:
		return Value{Type: ast.TypeBool, Data: false}, nil
	default:
		return Value{}, runtimeError(token, "unsupported literal "+token.Value)
	}
}

func zeroValue(valueType ast.ValueType) Value {
	switch valueType {
	case ast.TypeInt:
		return Value{Type: ast.TypeInt, Data: 0}
	case ast.TypeBool:
		return Value{Type: ast.TypeBool, Data: false}
	case ast.TypeString:
		return Value{Type: ast.TypeString, Data: ""}
	default:
		return Value{Type: ast.TypeUnknown, Data: nil}
	}
}

func valuesEqual(left, right Value) bool {
	if left.Type != right.Type {
		return false
	}

	switch left.Type {
	case ast.TypeInt:
		return left.Data.(int) == right.Data.(int)
	case ast.TypeBool:
		return left.Data.(bool) == right.Data.(bool)
	case ast.TypeString:
		return left.Data.(string) == right.Data.(string)
	default:
		return left.Data == right.Data
	}
}

func runtimeError(token tok.Token, message string) error {
	return RuntimeError{
		Message: message,
		Line:    token.Line,
		Column:  token.Column,
	}
}

func expressionToken(expression ast.Expr) tok.Token {
	switch expr := expression.(type) {
	case ast.LiteralExpr:
		return expr.Token
	case ast.VariableExpr:
		return expr.Name
	case ast.UnaryExpr:
		return expr.Operator
	case ast.BinaryExpr:
		return expr.Operator
	case ast.CallExpr:
		return expr.Paren
	case ast.AssignExpr:
		return expr.Name
	case ast.GroupingExpr:
		return expressionToken(expr.Expression)
	default:
		return tok.Token{}
	}
}
