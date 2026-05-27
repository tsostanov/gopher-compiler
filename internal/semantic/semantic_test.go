package semantic

import (
	"comp/internal/ast"
	"comp/internal/lexer"
	"comp/internal/options"
	"comp/internal/parser"
	tok "comp/internal/token"
	"strings"
	"testing"
)

func TestSemanticEnvironmentDefineAndResolveVariable(t *testing.T) {
	global := NewSemanticEnvironment(nil)
	token := tok.Token{Value: "x", Line: 1, Column: 5}

	variable, ok := global.DefineVariable(token, ast.TypeInt)
	if !ok {
		t.Fatalf("expected variable definition to succeed")
	}
	if variable == nil {
		t.Fatalf("expected variable info to be returned")
	}
	if !variable.Flags.Defined {
		t.Fatalf("expected variable to be marked as defined")
	}

	child := NewSemanticEnvironment(global)
	resolved := child.ResolveVariable("x")
	if resolved != variable {
		t.Fatalf("expected child environment to resolve parent variable")
	}
}

func TestSemanticEnvironmentRejectsSameScopeRedeclaration(t *testing.T) {
	environment := NewSemanticEnvironment(nil)
	token := tok.Token{Value: "x", Line: 1, Column: 5}

	if _, ok := environment.DefineVariable(token, ast.TypeInt); !ok {
		t.Fatalf("expected first declaration to succeed")
	}
	if _, ok := environment.DefineVariable(token, ast.TypeInt); ok {
		t.Fatalf("expected second declaration in same scope to fail")
	}
}

func TestSemanticAnalyzerUseBeforeInitialization(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int; print x;")

	assertHasDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerUndeclaredVariable(t *testing.T) {
	diagnostics := analyzeSource(t, "print x; x = 1;")

	assertHasDiagnostic(t, diagnostics, SeverityError, "use of undeclared variable x")
	assertHasDiagnostic(t, diagnostics, SeverityError, "assignment to undeclared variable x")
}

func TestSemanticAnalyzerInitializerMarksVariableInitialized(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int = 1; print x;")

	assertNoDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerAssignmentMarksVariableInitialized(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int; x = 1; print x;")

	assertNoDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerAllowsReassignmentWithSameType(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int = 1; x = 2; print x;")

	assertNoDiagnostic(t, diagnostics, SeverityError, "cannot assign")
}

func TestSemanticAnalyzerIfElsePropagatesInitialization(t *testing.T) {
	diagnostics := analyzeSource(t, `
var x: int;
if (true) {
	x = 1;
} else {
	x = 2;
}
print x;
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerIfWithoutElseDoesNotGuaranteeInitialization(t *testing.T) {
	diagnostics := analyzeSource(t, `
var x: int;
if (true) {
	x = 1;
}
print x;
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerWhileDoesNotGuaranteeInitialization(t *testing.T) {
	diagnostics := analyzeSource(t, `
var x: int;
while (false) {
	x = 1;
}
print x;
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerUnusedVariableWarning(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int = 1;")

	assertHasDiagnostic(t, diagnostics, SeverityWarning, "declared but never used")
}

func TestSemanticAnalyzerBlockScopeRedeclarationAllowed(t *testing.T) {
	diagnostics := analyzeSource(t, `
var x: int = 1;
{
	var x: int = 2;
	print x;
}
print x;
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "already declared in this scope")
}

func TestSemanticAnalyzerSameScopeRedeclarationRejected(t *testing.T) {
	diagnostics := analyzeSource(t, "var x: int = 1; var x: int = 2;")

	assertHasDiagnostic(t, diagnostics, SeverityError, "already declared in this scope")
}

func TestSemanticAnalyzerInfersTypeFromInitializer(t *testing.T) {
	diagnostics := analyzeSource(t, `var x = 1; x = 2; print x;`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "cannot assign")
}

func TestSemanticAnalyzerRejectsDeclarationWithoutTypeOrInitializer(t *testing.T) {
	diagnostics := analyzeSource(t, `var x;`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "requires an explicit type or initializer")
}

func TestSemanticAnalyzerAllowsCompatDeclarationWithoutInitializer(t *testing.T) {
	diagnostics := analyzeSourceWithOptions(t, `
var x;
x = 1;
print x;
`, options.Mode{CompatLoginov: true})

	assertNoDiagnostic(t, diagnostics, SeverityError, "requires an explicit type or initializer")
	assertNoDiagnostic(t, diagnostics, SeverityError, "used before initialization")
}

func TestSemanticAnalyzerRejectsMismatchedInitializerType(t *testing.T) {
	diagnostics := analyzeSource(t, `var x: int = "hello";`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "cannot initialize variable x of type int with value of type string")
}

func TestSemanticAnalyzerRejectsMismatchedAssignmentType(t *testing.T) {
	diagnostics := analyzeSource(t, `var x: bool = true; x = 1;`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "cannot assign value of type int to variable x of type bool")
}

func TestSemanticAnalyzerRequiresBooleanCondition(t *testing.T) {
	diagnostics := analyzeSource(t, `if (1) { print 1; }`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "if condition must have type bool")
}

func TestSemanticAnalyzerChecksBinaryOperatorTypes(t *testing.T) {
	diagnostics := analyzeSource(t, `print "a" - "b"; print true and 1;`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "arithmetic operators expect operands of type int")
	assertHasDiagnostic(t, diagnostics, SeverityError, "logical operators expect operands of type bool")
}

func TestSemanticAnalyzerAllowsStringConcatenation(t *testing.T) {
	diagnostics := analyzeSource(t, `var s: string = "a" + "b"; print s;`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "operator +")
}

func TestSemanticAnalyzerRejectsIntAndStringAddition(t *testing.T) {
	diagnostics := analyzeSource(t, `print 1 + "a"; print "a" + 1;`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "operator + expects operands of type int or string")
}

func TestSemanticAnalyzerAllowsStringEqualityComparisons(t *testing.T) {
	diagnostics := analyzeSource(t, `
if ("a" == "b") {
	print "eq";
}
if ("a" != "b") {
	print "neq";
}
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "equality operators require operands of the same type")
	assertNoDiagnostic(t, diagnostics, SeverityError, "if condition must have type bool")
}

func TestSemanticAnalyzerRejectsStringOrderingComparisons(t *testing.T) {
	diagnostics := analyzeSource(t, `print "a" < "b"; print "a" >= "b";`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "comparison operators expect operands of type int")
}

func TestSemanticAnalyzerRejectsStringConditionInIf(t *testing.T) {
	diagnostics := analyzeSource(t, `if ("a" + "b") { print 1; }`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "if condition must have type bool")
}

func TestSemanticAnalyzerAllowsFunctionDeclarationAndCall(t *testing.T) {
	diagnostics := analyzeSource(t, `
func add(a: int, b: int): int {
	return a + b;
}

print add(1, 2);
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "call to undeclared function")
	assertNoDiagnostic(t, diagnostics, SeverityError, "cannot return")
}

func TestSemanticAnalyzerAllowsCallBeforeFunctionDeclaration(t *testing.T) {
	diagnostics := analyzeSource(t, `
print twice(3);

func twice(value: int): int {
	return value * 2;
}
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "call to undeclared function")
}

func TestSemanticAnalyzerRejectsWrongFunctionArgumentType(t *testing.T) {
	diagnostics := analyzeSource(t, `
func negate(flag: bool): bool {
	return !flag;
}

print negate(1);
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "cannot pass value of type int to parameter flag of type bool")
}

func TestSemanticAnalyzerRejectsReturnOutsideFunction(t *testing.T) {
	diagnostics := analyzeSource(t, `return 1;`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "return statement is only allowed inside functions")
}

func TestSemanticAnalyzerAllowsReturnWithoutValueInUntypedFunction(t *testing.T) {
	diagnostics := analyzeSource(t, `
fun noop() {
	return;
}
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "cannot return")
}

func TestSemanticAnalyzerRejectsReturnWithoutValueInTypedFunction(t *testing.T) {
	diagnostics := analyzeSource(t, `
func bad(): int {
	return;
}
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "cannot return without value")
}

func TestSemanticAnalyzerRejectsFunctionWithoutGuaranteedReturn(t *testing.T) {
	diagnostics := analyzeSource(t, `
func maybe(value: int): int {
	if (value > 0) {
		return value;
	}
}
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "may not return a value on all paths")
}

func TestSemanticAnalyzerWarnsAboutDeadIfBranch(t *testing.T) {
	diagnostics := analyzeSource(t, `
if (1 == 2) {
	print "never";
}
`)

	assertHasDiagnostic(t, diagnostics, SeverityWarning, "then branch never executes")
}

func TestSemanticAnalyzerWarnsAboutDeadWhileBody(t *testing.T) {
	diagnostics := analyzeSource(t, `
while (false) {
	print "never";
}
`)

	assertHasDiagnostic(t, diagnostics, SeverityWarning, "while body never executes")
}

func TestSemanticAnalyzerSupportsArrays(t *testing.T) {
	diagnostics := analyzeSource(t, `
var xs: int[] = [1, 2, 3];
xs[1] = xs[0];
print xs[2];
`)

	assertNoDiagnostic(t, diagnostics, SeverityError, "array")
	assertNoDiagnostic(t, diagnostics, SeverityError, "index")
}

func TestSemanticAnalyzerRejectsMixedArrayElementTypes(t *testing.T) {
	diagnostics := analyzeSource(t, `var xs = [1, true];`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "array elements must have the same type")
}

func TestSemanticAnalyzerRejectsNonIntArrayIndex(t *testing.T) {
	diagnostics := analyzeSource(t, `
var xs = [1, 2];
print xs[true];
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "array index must have type int")
}

func TestSemanticAnalyzerRejectsIndexingNonArray(t *testing.T) {
	diagnostics := analyzeSource(t, `
var x = 1;
print x[0];
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "indexing requires an array value")
}

func TestSemanticAnalyzerRejectsWrongArrayElementAssignmentType(t *testing.T) {
	diagnostics := analyzeSource(t, `
var xs: int[] = [1];
xs[0] = false;
`)

	assertHasDiagnostic(t, diagnostics, SeverityError, "cannot assign value of type bool to array element of type int")
}

func analyzeSource(t *testing.T, source string) []SemanticDiagnostic {
	t.Helper()

	return analyzeSourceWithOptions(t, source, options.Mode{})
}

func analyzeSourceWithOptions(t *testing.T, source string, mode options.Mode) []SemanticDiagnostic {
	t.Helper()

	lexer := lexer.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	parser := parser.NewParserWithOptions(tokens, mode)
	statements, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := NewSemanticAnalyzerWithOptions(mode)
	return analyzer.Analyze(statements)
}

func assertHasDiagnostic(t *testing.T, diagnostics []SemanticDiagnostic, severity DiagnosticSeverity, fragment string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == severity && strings.Contains(diagnostic.Message, fragment) {
			return
		}
	}

	t.Fatalf("expected %s containing %q, got %#v", severity, fragment, diagnostics)
}

func assertNoDiagnostic(t *testing.T, diagnostics []SemanticDiagnostic, severity DiagnosticSeverity, fragment string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == severity && strings.Contains(diagnostic.Message, fragment) {
			t.Fatalf("unexpected %s containing %q: %#v", severity, fragment, diagnostic)
		}
	}
}
