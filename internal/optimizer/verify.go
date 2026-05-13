package optimizer

import (
	"bytes"
	"comp/internal/ast"
	"comp/internal/executor"
	"comp/internal/options"
	"comp/internal/semantic"
	"strings"
)

type VerificationResult struct {
	Ok              bool
	OriginalOutput  string
	OptimizedOutput string
	OriginalError   error
	OptimizedError  error
	Message         string
}

type OptimizationVerifier struct {
	mode options.Mode
}

func NewOptimizationVerifier() *OptimizationVerifier {
	return NewOptimizationVerifierWithOptions(options.Mode{})
}

func NewOptimizationVerifierWithOptions(mode options.Mode) *OptimizationVerifier {
	return &OptimizationVerifier{mode: mode}
}

func (v *OptimizationVerifier) Verify(original []ast.Stmt, optimized []ast.Stmt) VerificationResult {
	if diagnostics := analyzeSemantics(original, v.mode); hasSemanticErrors(diagnostics) {
		return VerificationResult{
			Ok:      false,
			Message: "original AST has semantic errors: " + joinDiagnostics(diagnostics),
		}
	}

	if diagnostics := analyzeSemantics(optimized, v.mode); hasSemanticErrors(diagnostics) {
		return VerificationResult{
			Ok:      false,
			Message: "optimized AST has semantic errors: " + joinDiagnostics(diagnostics),
		}
	}

	var originalOutput bytes.Buffer
	var optimizedOutput bytes.Buffer

	originalExecutor := executor.NewExecutorWithOptions(&originalOutput, v.mode)
	optimizedExecutor := executor.NewExecutorWithOptions(&optimizedOutput, v.mode)

	originalErr := originalExecutor.Execute(original)
	optimizedErr := optimizedExecutor.Execute(optimized)

	result := VerificationResult{
		OriginalOutput:  originalOutput.String(),
		OptimizedOutput: optimizedOutput.String(),
		OriginalError:   originalErr,
		OptimizedError:  optimizedErr,
	}

	if result.OriginalOutput != result.OptimizedOutput {
		result.Message = "optimization changed program output"
		return result
	}

	if !sameError(originalErr, optimizedErr) {
		result.Message = "optimization changed runtime behavior"
		return result
	}

	result.Ok = true
	result.Message = "optimized AST is equivalent to original AST"
	return result
}

func analyzeSemantics(statements []ast.Stmt, mode options.Mode) []semantic.SemanticDiagnostic {
	analyzer := semantic.NewSemanticAnalyzerWithOptions(mode)
	return analyzer.Analyze(statements)
}

func hasSemanticErrors(diagnostics []semantic.SemanticDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == semantic.SeverityError {
			return true
		}
	}
	return false
}

func joinDiagnostics(diagnostics []semantic.SemanticDiagnostic) string {
	parts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != semantic.SeverityError {
			continue
		}
		parts = append(parts, diagnostic.String())
	}
	return strings.Join(parts, "; ")
}

func sameError(left, right error) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Error() == right.Error()
}
