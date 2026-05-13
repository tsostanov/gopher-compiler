package semantic

import (
	"comp/internal/ast"
	"comp/internal/options"
	tok "comp/internal/token"
	"fmt"
)

type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
)

type SemanticDiagnostic struct {
	Severity DiagnosticSeverity
	Message  string
	Line     int
	Column   int
}

func (d SemanticDiagnostic) String() string {
	return fmt.Sprintf("%s at %d:%d: %s", d.Severity, d.Line, d.Column, d.Message)
}

type SymbolFlags struct {
	Defined     bool
	Initialized bool
	Used        bool
}

type VariableInfo struct {
	Name       string
	DeclaredAt tok.Token
	Type       ast.ValueType
	Flags      SymbolFlags
}

type ParameterInfo struct {
	Name       string
	DeclaredAt tok.Token
	Type       ast.ValueType
}

type FunctionInfo struct {
	Name       string
	DeclaredAt tok.Token
	ReturnType ast.ValueType
	Parameters []ParameterInfo
}

type SemanticEnvironment struct {
	parent    *SemanticEnvironment
	variables map[string]*VariableInfo
	functions map[string]*FunctionInfo
}

func NewSemanticEnvironment(parent *SemanticEnvironment) *SemanticEnvironment {
	return &SemanticEnvironment{
		parent:    parent,
		variables: make(map[string]*VariableInfo),
		functions: make(map[string]*FunctionInfo),
	}
}

func (e *SemanticEnvironment) Parent() *SemanticEnvironment {
	return e.parent
}

func (e *SemanticEnvironment) DefineVariable(name tok.Token, variableType ast.ValueType) (*VariableInfo, bool) {
	if _, exists := e.variables[name.Value]; exists {
		return nil, false
	}
	if _, exists := e.functions[name.Value]; exists {
		return nil, false
	}

	variable := &VariableInfo{
		Name:       name.Value,
		DeclaredAt: name,
		Type:       variableType,
		Flags: SymbolFlags{
			Defined: true,
		},
	}
	e.variables[name.Value] = variable
	return variable, true
}

func (e *SemanticEnvironment) ResolveVariable(name string) *VariableInfo {
	for current := e; current != nil; current = current.parent {
		if variable, ok := current.variables[name]; ok {
			return variable
		}
	}
	return nil
}

func (e *SemanticEnvironment) DefineFunction(name tok.Token, returnType ast.ValueType, parameters []ParameterInfo) (*FunctionInfo, bool) {
	if _, exists := e.functions[name.Value]; exists {
		return nil, false
	}
	if _, exists := e.variables[name.Value]; exists {
		return nil, false
	}

	function := &FunctionInfo{
		Name:       name.Value,
		DeclaredAt: name,
		ReturnType: returnType,
		Parameters: parameters,
	}
	e.functions[name.Value] = function
	return function, true
}

func (e *SemanticEnvironment) ResolveFunction(name string) *FunctionInfo {
	for current := e; current != nil; current = current.parent {
		if function, ok := current.functions[name]; ok {
			return function
		}
	}
	return nil
}

func (e *SemanticEnvironment) ResolveFunctionInCurrentScope(name string) *FunctionInfo {
	return e.functions[name]
}

type variableState struct {
	defined     bool
	initialized bool
	used        bool
}

type constantValue struct {
	valueType ast.ValueType
	data      any
}

type SemanticAnalyzer struct {
	environment     *SemanticEnvironment
	variables       []*VariableInfo
	diagnostics     []SemanticDiagnostic
	currentFunction *FunctionInfo
	options         options.Mode
}

func NewSemanticAnalyzer() *SemanticAnalyzer {
	return NewSemanticAnalyzerWithOptions(options.Mode{})
}

func NewSemanticAnalyzerWithOptions(mode options.Mode) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		environment: NewSemanticEnvironment(nil),
		options:     mode,
	}
}

func (a *SemanticAnalyzer) Analyze(statements []ast.Stmt) []SemanticDiagnostic {
	a.environment = NewSemanticEnvironment(nil)
	a.variables = nil
	a.diagnostics = nil
	a.currentFunction = nil
	a.analyzeStatements(statements)
	a.reportUnusedVariables()
	return a.diagnostics
}

func (a *SemanticAnalyzer) analyzeStatements(statements []ast.Stmt) {
	a.predeclareFunctions(statements)
	for _, statement := range statements {
		a.VisitStatement(statement)
	}
}

func (a *SemanticAnalyzer) predeclareFunctions(statements []ast.Stmt) {
	for _, statement := range statements {
		function, ok := statement.(ast.FuncStmt)
		if !ok {
			continue
		}
		a.defineFunctionSignature(function)
	}
}

func (a *SemanticAnalyzer) defineFunctionSignature(statement ast.FuncStmt) *FunctionInfo {
	parameters := make([]ParameterInfo, 0, len(statement.Parameters))
	for _, parameter := range statement.Parameters {
		parameters = append(parameters, ParameterInfo{
			Name:       parameter.Name.Value,
			DeclaredAt: parameter.Name,
			Type:       parameter.Type.Kind,
		})
	}

	function, ok := a.environment.DefineFunction(statement.Name, statement.ReturnType.Kind, parameters)
	if !ok {
		a.errorAt(statement.Name, "name "+statement.Name.Value+" is already declared in this scope")
		return nil
	}
	return function
}

func (a *SemanticAnalyzer) VisitStatement(statement ast.Stmt) {
	switch s := statement.(type) {
	case ast.VarStmt:
		a.visitVarStatement(s)
	case ast.FuncStmt:
		function := a.environment.ResolveFunctionInCurrentScope(s.Name.Value)
		if function == nil {
			function = a.defineFunctionSignature(s)
		}
		if function == nil {
			return
		}
		a.visitFunctionBody(s, function)
	case ast.ReturnStmt:
		a.visitReturnStatement(s)
	case ast.PrintStmt:
		a.VisitExpression(s.Expression)
	case ast.ExprStmt:
		a.VisitExpression(s.Expression)
	case ast.BlockStmt:
		previousEnvironment := a.environment
		a.environment = NewSemanticEnvironment(previousEnvironment)
		a.analyzeStatements(s.Statements)
		a.environment = previousEnvironment
	case ast.IfStmt:
		conditionType := a.VisitExpression(s.Condition)
		a.requireType(conditionType, ast.TypeBool, s.Condition, "if condition must have type bool")
		if always, value := a.constantBoolValue(s.Condition); always && !value {
			a.warningAt(s.Keyword, "unreachable code: then branch never executes")
		}

		before := a.snapshotVariableStates()
		a.VisitStatement(s.ThenBranch)
		thenState := a.snapshotVariableStates()

		if s.ElseBranch == nil {
			a.restoreVariableStates(before)
			a.mergeVariableStates(before, thenState, nil)
			return
		}

		a.restoreVariableStates(before)
		a.VisitStatement(s.ElseBranch)
		elseState := a.snapshotVariableStates()
		a.restoreVariableStates(before)
		a.mergeVariableStates(before, thenState, elseState)
	case ast.WhileStmt:
		conditionType := a.VisitExpression(s.Condition)
		a.requireType(conditionType, ast.TypeBool, s.Condition, "while condition must have type bool")
		if always, value := a.constantBoolValue(s.Condition); always && !value {
			a.warningAt(s.Keyword, "unreachable code: while body never executes")
		}

		before := a.snapshotVariableStates()
		a.VisitStatement(s.Body)
		bodyState := a.snapshotVariableStates()
		a.restoreVariableStates(before)

		for variable, state := range before {
			body := bodyState[variable]
			variable.Flags.Defined = state.defined
			variable.Flags.Initialized = state.initialized
			variable.Flags.Used = state.used || body.used
		}
	}
}

func (a *SemanticAnalyzer) visitVarStatement(statement ast.VarStmt) {
	declaredType := ast.TypeUnknown
	if statement.DeclaredType != nil {
		declaredType = statement.DeclaredType.Kind
	}

	initializerType := ast.TypeUnknown
	if statement.Initializer != nil {
		initializerType = a.VisitExpression(statement.Initializer)
	}

	variableType := declaredType
	if variableType == ast.TypeUnknown {
		variableType = initializerType
	}

	variable, ok := a.environment.DefineVariable(statement.Name, variableType)
	if !ok {
		a.errorAt(statement.Name, "name "+statement.Name.Value+" is already declared in this scope")
		return
	}
	a.variables = append(a.variables, variable)

	if statement.DeclaredType == nil && statement.Initializer == nil && !a.options.CompatLoginov {
		a.errorAt(statement.Name, "variable "+statement.Name.Value+" requires an explicit type or initializer")
	}
	if declaredType != ast.TypeUnknown && initializerType != ast.TypeUnknown && !a.isAssignable(declaredType, initializerType) {
		a.errorAt(statement.Name, "cannot initialize variable "+statement.Name.Value+" of type "+declaredType.String()+" with value of type "+initializerType.String())
	}
	if statement.Initializer != nil {
		variable.Flags.Initialized = true
	}
}

func (a *SemanticAnalyzer) visitFunctionBody(statement ast.FuncStmt, function *FunctionInfo) {
	previousEnvironment := a.environment
	previousFunction := a.currentFunction

	a.environment = NewSemanticEnvironment(previousEnvironment)
	a.currentFunction = function
	defer func() {
		a.environment = previousEnvironment
		a.currentFunction = previousFunction
	}()

	for _, parameter := range statement.Parameters {
		variable, ok := a.environment.DefineVariable(parameter.Name, parameter.Type.Kind)
		if !ok {
			a.errorAt(parameter.Name, "name "+parameter.Name.Value+" is already declared in this scope")
			continue
		}
		variable.Flags.Initialized = true
		a.variables = append(a.variables, variable)
	}

	a.analyzeStatements(statement.Body.Statements)

	if statement.ReturnType.Kind != ast.TypeUnknown && !guaranteesReturnBlock(statement.Body) {
		a.errorAt(statement.Name, "function "+statement.Name.Value+" may not return a value on all paths")
	}
}

func (a *SemanticAnalyzer) visitReturnStatement(statement ast.ReturnStmt) {
	if a.currentFunction == nil {
		a.errorAt(statement.Keyword, "return statement is only allowed inside functions")
		return
	}

	if statement.Value == nil {
		if a.currentFunction.ReturnType != ast.TypeUnknown {
			a.errorAt(statement.Keyword, "cannot return without value from function "+a.currentFunction.Name+" with return type "+a.currentFunction.ReturnType.String())
		}
		return
	}

	valueType := a.VisitExpression(statement.Value)
	if !a.isAssignable(a.currentFunction.ReturnType, valueType) {
		a.errorAt(statement.Keyword, "cannot return value of type "+valueType.String()+" from function "+a.currentFunction.Name+" with return type "+a.currentFunction.ReturnType.String())
	}
}

func (a *SemanticAnalyzer) VisitExpression(expression ast.Expr) ast.ValueType {
	switch e := expression.(type) {
	case ast.LiteralExpr:
		return literalType(e.Token)
	case ast.VariableExpr:
		variable := a.environment.ResolveVariable(e.Name.Value)
		if variable == nil || !variable.Flags.Defined {
			a.errorAt(e.Name, "use of undeclared variable "+e.Name.Value)
			return ast.TypeUnknown
		}

		variable.Flags.Used = true
		if !variable.Flags.Initialized {
			a.errorAt(e.Name, "variable "+e.Name.Value+" is used before initialization")
		}
		return variable.Type
	case ast.AssignExpr:
		valueType := a.VisitExpression(e.Value)

		variable := a.environment.ResolveVariable(e.Name.Value)
		if variable == nil || !variable.Flags.Defined {
			a.errorAt(e.Name, "assignment to undeclared variable "+e.Name.Value)
			return ast.TypeUnknown
		}
		if variable.Type != ast.TypeUnknown && valueType != ast.TypeUnknown && !a.isAssignable(variable.Type, valueType) {
			a.errorAt(e.Name, "cannot assign value of type "+valueType.String()+" to variable "+e.Name.Value+" of type "+variable.Type.String())
		}
		if variable.Type == ast.TypeUnknown && valueType != ast.TypeUnknown {
			variable.Type = valueType
		}

		variable.Flags.Initialized = true
		return variable.Type
	case ast.BinaryExpr:
		leftType := a.VisitExpression(e.Left)
		rightType := a.VisitExpression(e.Right)
		return a.checkBinaryExpression(e, leftType, rightType)
	case ast.UnaryExpr:
		rightType := a.VisitExpression(e.Right)
		return a.checkUnaryExpression(e, rightType)
	case ast.GroupingExpr:
		return a.VisitExpression(e.Expression)
	case ast.CallExpr:
		return a.visitCallExpression(e)
	}
	return ast.TypeUnknown
}

func (a *SemanticAnalyzer) visitCallExpression(expression ast.CallExpr) ast.ValueType {
	callee, ok := expression.Callee.(ast.VariableExpr)
	if !ok {
		a.errorAt(expression.Paren, "can only call named functions")
		for _, argument := range expression.Arguments {
			a.VisitExpression(argument)
		}
		return ast.TypeUnknown
	}

	function := a.environment.ResolveFunction(callee.Name.Value)
	if function == nil {
		a.errorAt(callee.Name, "call to undeclared function "+callee.Name.Value)
		for _, argument := range expression.Arguments {
			a.VisitExpression(argument)
		}
		return ast.TypeUnknown
	}

	if len(expression.Arguments) != len(function.Parameters) {
		a.errorAt(callee.Name, fmt.Sprintf("function %s expects %d arguments, got %d", function.Name, len(function.Parameters), len(expression.Arguments)))
	}

	for index, argument := range expression.Arguments {
		argumentType := a.VisitExpression(argument)
		if index >= len(function.Parameters) {
			continue
		}
		parameter := function.Parameters[index]
		if !a.isAssignable(parameter.Type, argumentType) {
			a.errorAt(expressionToken(argument), "cannot pass value of type "+argumentType.String()+" to parameter "+parameter.Name+" of type "+parameter.Type.String())
		}
	}

	return function.ReturnType
}

func (a *SemanticAnalyzer) HasErrors() bool {
	for _, diagnostic := range a.diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (a *SemanticAnalyzer) Diagnostics() []SemanticDiagnostic {
	return a.diagnostics
}

func (a *SemanticAnalyzer) snapshotVariableStates() map[*VariableInfo]variableState {
	snapshot := make(map[*VariableInfo]variableState, len(a.variables))
	for _, variable := range a.variables {
		snapshot[variable] = variableState{
			defined:     variable.Flags.Defined,
			initialized: variable.Flags.Initialized,
			used:        variable.Flags.Used,
		}
	}
	return snapshot
}

func (a *SemanticAnalyzer) restoreVariableStates(snapshot map[*VariableInfo]variableState) {
	for variable, state := range snapshot {
		variable.Flags.Defined = state.defined
		variable.Flags.Initialized = state.initialized
		variable.Flags.Used = state.used
	}
}

func (a *SemanticAnalyzer) mergeVariableStates(before, left, right map[*VariableInfo]variableState) {
	for variable, state := range before {
		leftState := left[variable]
		if right == nil {
			variable.Flags.Defined = state.defined
			variable.Flags.Initialized = state.initialized
			variable.Flags.Used = state.used || leftState.used
			continue
		}

		rightState := right[variable]
		variable.Flags.Defined = leftState.defined && rightState.defined
		variable.Flags.Initialized = leftState.initialized && rightState.initialized
		variable.Flags.Used = leftState.used || rightState.used
	}
}

func (a *SemanticAnalyzer) reportUnusedVariables() {
	for _, variable := range a.variables {
		if !variable.Flags.Defined || variable.Flags.Used {
			continue
		}

		a.diagnostics = append(a.diagnostics, SemanticDiagnostic{
			Severity: SeverityWarning,
			Message:  "variable " + variable.Name + " is declared but never used",
			Line:     variable.DeclaredAt.Line,
			Column:   variable.DeclaredAt.Column,
		})
	}
}

func (a *SemanticAnalyzer) errorAt(token tok.Token, message string) {
	a.diagnostics = append(a.diagnostics, SemanticDiagnostic{
		Severity: SeverityError,
		Message:  message,
		Line:     token.Line,
		Column:   token.Column,
	})
}

func (a *SemanticAnalyzer) warningAt(token tok.Token, message string) {
	a.diagnostics = append(a.diagnostics, SemanticDiagnostic{
		Severity: SeverityWarning,
		Message:  message,
		Line:     token.Line,
		Column:   token.Column,
	})
}

func (a *SemanticAnalyzer) requireType(actual, expected ast.ValueType, expression ast.Expr, message string) {
	if actual == ast.TypeUnknown || actual == expected {
		return
	}
	a.errorAt(expressionToken(expression), message)
}

func (a *SemanticAnalyzer) isAssignable(target, value ast.ValueType) bool {
	if target == ast.TypeUnknown || value == ast.TypeUnknown {
		return true
	}
	return target == value
}

func (a *SemanticAnalyzer) checkUnaryExpression(expression ast.UnaryExpr, rightType ast.ValueType) ast.ValueType {
	switch expression.Operator.Type {
	case tok.TokenMinus:
		if rightType != ast.TypeUnknown && rightType != ast.TypeInt {
			a.errorAt(expression.Operator, "operator - expects operand of type int")
		}
		return ast.TypeInt
	case tok.TokenExcl:
		if rightType != ast.TypeUnknown && rightType != ast.TypeBool {
			a.errorAt(expression.Operator, "operator ! expects operand of type bool")
		}
		return ast.TypeBool
	default:
		return ast.TypeUnknown
	}
}

func (a *SemanticAnalyzer) checkBinaryExpression(expression ast.BinaryExpr, leftType, rightType ast.ValueType) ast.ValueType {
	switch expression.Operator.Type {
	case tok.TokenPlus:
		if leftType == ast.TypeUnknown || rightType == ast.TypeUnknown {
			return ast.TypeUnknown
		}
		if leftType == ast.TypeInt && rightType == ast.TypeInt {
			return ast.TypeInt
		}
		if leftType == ast.TypeString && rightType == ast.TypeString {
			return ast.TypeString
		}
		a.errorAt(expression.Operator, "operator + expects operands of type int or string")
		return ast.TypeUnknown
	case tok.TokenMinus, tok.TokenStar, tok.TokenSlash:
		if leftType != ast.TypeUnknown && leftType != ast.TypeInt {
			a.errorAt(expression.Operator, "arithmetic operators expect operands of type int")
		}
		if rightType != ast.TypeUnknown && rightType != ast.TypeInt {
			a.errorAt(expression.Operator, "arithmetic operators expect operands of type int")
		}
		return ast.TypeInt
	case tok.TokenAnd, tok.TokenOr:
		if leftType != ast.TypeUnknown && leftType != ast.TypeBool {
			a.errorAt(expression.Operator, "logical operators expect operands of type bool")
		}
		if rightType != ast.TypeUnknown && rightType != ast.TypeBool {
			a.errorAt(expression.Operator, "logical operators expect operands of type bool")
		}
		return ast.TypeBool
	case tok.TokenLt, tok.TokenLtEq, tok.TokenGt, tok.TokenGtEq:
		if leftType != ast.TypeUnknown && leftType != ast.TypeInt {
			a.errorAt(expression.Operator, "comparison operators expect operands of type int")
		}
		if rightType != ast.TypeUnknown && rightType != ast.TypeInt {
			a.errorAt(expression.Operator, "comparison operators expect operands of type int")
		}
		return ast.TypeBool
	case tok.TokenEqEq, tok.TokenNeq:
		if leftType != ast.TypeUnknown && rightType != ast.TypeUnknown && leftType != rightType {
			a.errorAt(expression.Operator, "equality operators require operands of the same type")
		}
		return ast.TypeBool
	default:
		return ast.TypeUnknown
	}
}

func (a *SemanticAnalyzer) constantBoolValue(expression ast.Expr) (bool, bool) {
	value, ok := a.constantValue(expression)
	if !ok || value.valueType != ast.TypeBool {
		return false, false
	}
	return true, value.data.(bool)
}

func (a *SemanticAnalyzer) constantValue(expression ast.Expr) (constantValue, bool) {
	switch e := expression.(type) {
	case ast.LiteralExpr:
		switch e.Token.Type {
		case tok.TokenNumber:
			return constantValue{valueType: ast.TypeInt, data: mustAtoi(e.Token.Value)}, true
		case tok.TokenString:
			return constantValue{valueType: ast.TypeString, data: e.Token.Value}, true
		case tok.TokenTrue:
			return constantValue{valueType: ast.TypeBool, data: true}, true
		case tok.TokenFalse:
			return constantValue{valueType: ast.TypeBool, data: false}, true
		}
	case ast.GroupingExpr:
		return a.constantValue(e.Expression)
	case ast.UnaryExpr:
		right, ok := a.constantValue(e.Right)
		if !ok {
			return constantValue{}, false
		}
		switch e.Operator.Type {
		case tok.TokenExcl:
			if right.valueType != ast.TypeBool {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: !right.data.(bool)}, true
		case tok.TokenMinus:
			if right.valueType != ast.TypeInt {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeInt, data: -right.data.(int)}, true
		}
	case ast.BinaryExpr:
		left, ok := a.constantValue(e.Left)
		if !ok {
			return constantValue{}, false
		}
		right, ok := a.constantValue(e.Right)
		if !ok {
			return constantValue{}, false
		}

		switch e.Operator.Type {
		case tok.TokenEqEq:
			if left.valueType != right.valueType {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data == right.data}, true
		case tok.TokenNeq:
			if left.valueType != right.valueType {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data != right.data}, true
		case tok.TokenLt:
			if left.valueType != ast.TypeInt || right.valueType != ast.TypeInt {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data.(int) < right.data.(int)}, true
		case tok.TokenLtEq:
			if left.valueType != ast.TypeInt || right.valueType != ast.TypeInt {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data.(int) <= right.data.(int)}, true
		case tok.TokenGt:
			if left.valueType != ast.TypeInt || right.valueType != ast.TypeInt {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data.(int) > right.data.(int)}, true
		case tok.TokenGtEq:
			if left.valueType != ast.TypeInt || right.valueType != ast.TypeInt {
				return constantValue{}, false
			}
			return constantValue{valueType: ast.TypeBool, data: left.data.(int) >= right.data.(int)}, true
		}
	}

	return constantValue{}, false
}

func guaranteesReturnBlock(block ast.BlockStmt) bool {
	for _, statement := range block.Statements {
		if guaranteesReturnStatement(statement) {
			return true
		}
	}
	return false
}

func guaranteesReturnStatement(statement ast.Stmt) bool {
	switch s := statement.(type) {
	case ast.ReturnStmt:
		return true
	case ast.BlockStmt:
		return guaranteesReturnBlock(s)
	case ast.IfStmt:
		return s.ElseBranch != nil && guaranteesReturnStatement(s.ThenBranch) && guaranteesReturnStatement(s.ElseBranch)
	default:
		return false
	}
}

func literalType(token tok.Token) ast.ValueType {
	switch token.Type {
	case tok.TokenNumber:
		return ast.TypeInt
	case tok.TokenString:
		return ast.TypeString
	case tok.TokenTrue, tok.TokenFalse:
		return ast.TypeBool
	default:
		return ast.TypeUnknown
	}
}

func expressionToken(expression ast.Expr) tok.Token {
	switch e := expression.(type) {
	case ast.LiteralExpr:
		return e.Token
	case ast.VariableExpr:
		return e.Name
	case ast.UnaryExpr:
		return e.Operator
	case ast.BinaryExpr:
		return e.Operator
	case ast.CallExpr:
		return e.Paren
	case ast.AssignExpr:
		return e.Name
	case ast.GroupingExpr:
		return expressionToken(e.Expression)
	default:
		return tok.Token{}
	}
}

func mustAtoi(value string) int {
	var result int
	for i := 0; i < len(value); i++ {
		result = result*10 + int(value[i]-'0')
	}
	return result
}
