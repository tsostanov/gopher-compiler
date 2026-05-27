package executor

import (
	"comp/internal/options"
	"strings"
	"testing"
)

func TestExecutorPrintsArithmeticResult(t *testing.T) {
	output, err := executeSource(t, "print 1 + 2 * 3;")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "7\n" {
		t.Fatalf("expected output %q, got %q", "7\n", output)
	}
}

func TestExecutorSupportsBlockScope(t *testing.T) {
	output, err := executeSource(t, `
var x: int = 1;
{
	var x: int = 2;
	print x;
}
print x;
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "2\n1\n" {
		t.Fatalf("expected output %q, got %q", "2\n1\n", output)
	}
}

func TestExecutorExecutesIfElse(t *testing.T) {
	output, err := executeSource(t, `
var x: int;
if (true) {
	x = 10;
} else {
	x = 20;
}
print x;
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "10\n" {
		t.Fatalf("expected output %q, got %q", "10\n", output)
	}
}

func TestExecutorExecutesWhileLoop(t *testing.T) {
	output, err := executeSource(t, `
var x: int = 0;
while (x < 3) {
	print x;
	x = x + 1;
}
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "0\n1\n2\n" {
		t.Fatalf("expected output %q, got %q", "0\n1\n2\n", output)
	}
}

func TestExecutorSupportsStringsAndEquality(t *testing.T) {
	output, err := executeSource(t, `
print "a" + "b";
print "a" == "a";
print "a" != "b";
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "ab\ntrue\ntrue\n" {
		t.Fatalf("expected output %q, got %q", "ab\ntrue\ntrue\n", output)
	}
}

func TestExecutorReportsDivisionByZero(t *testing.T) {
	_, err := executeSource(t, `print 1 / 0;`)
	if err == nil {
		t.Fatalf("expected runtime error")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("expected division by zero error, got %v", err)
	}
}

func TestExecutorCallsFunction(t *testing.T) {
	output, err := executeSource(t, `
func add(a: int, b: int): int {
	return a + b;
}

print add(2, 5);
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "7\n" {
		t.Fatalf("expected output %q, got %q", "7\n", output)
	}
}

func TestExecutorAllowsCallBeforeFunctionDeclaration(t *testing.T) {
	output, err := executeSource(t, `
print twice(4);

func twice(value: int): int {
	return value * 2;
}
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "8\n" {
		t.Fatalf("expected output %q, got %q", "8\n", output)
	}
}

func TestExecutorSupportsRecursion(t *testing.T) {
	output, err := executeSource(t, `
func fact(n: int): int {
	if (n == 0) {
		return 1;
	}
	return n * fact(n - 1);
}

print fact(5);
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "120\n" {
		t.Fatalf("expected output %q, got %q", "120\n", output)
	}
}

func TestExecutorSupportsUntypedFunSyntax(t *testing.T) {
	output, err := executeSource(t, `
fun add(a, b) {
	return a + b;
}

var result = add(5, 10);
print result;
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "15\n" {
		t.Fatalf("expected output %q, got %q", "15\n", output)
	}
}

func TestExecutorSupportsCompatVarWithoutInitializer(t *testing.T) {
	output, err := executeSourceWithOptions(t, `
var x;
x = 10;
print x;
`, options.Mode{CompatLoginov: true})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "10\n" {
		t.Fatalf("expected output %q, got %q", "10\n", output)
	}
}

func TestExecutorAllowsReturnWithoutValue(t *testing.T) {
	output, err := executeSource(t, `
fun noop() {
	return;
}

noop();
print "ok";
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "ok\n" {
		t.Fatalf("expected output %q, got %q", "ok\n", output)
	}
}

func TestExecutorSupportsArrays(t *testing.T) {
	output, err := executeSource(t, `
var xs: int[] = [1, 2, 3];
xs[1] = xs[0] + xs[2];
print xs[1];
print xs[0];
`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if output != "4\n1\n" {
		t.Fatalf("expected output %q, got %q", "4\n1\n", output)
	}
}

func TestExecutorReportsArrayIndexOutOfBounds(t *testing.T) {
	_, err := executeSource(t, `
var xs = [1, 2];
print xs[2];
`)
	if err == nil {
		t.Fatalf("expected runtime error")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Fatalf("expected out of bounds error, got %v", err)
	}
}
