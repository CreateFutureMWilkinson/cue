package sqlite_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/suite"
)

// GosecComplianceSuite verifies that source files handle db.Close() return
// values explicitly, satisfying gosec G104 (CWE-703).
type GosecComplianceSuite struct {
	suite.Suite
}

func TestGosecCompliance(t *testing.T) {
	suite.Run(t, new(GosecComplianceSuite))
}

// findBareCloseCalls parses the given Go source file and returns the line
// numbers of any db.Close() calls whose return value is not explicitly
// handled (i.e., not assigned to _ or a variable). A bare expression
// statement like `db.Close()` is flagged; `_ = db.Close()` is not.
func (s *GosecComplianceSuite) findBareCloseCalls(filePath string) []int {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	s.Require().NoError(err, "failed to parse %s", filePath)

	var bareLines []int

	ast.Inspect(f, func(n ast.Node) bool {
		exprStmt, ok := n.(*ast.ExprStmt)
		if !ok {
			return true
		}

		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "Close" {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "db" {
				bareLines = append(bareLines, fset.Position(call.Pos()).Line)
			}
		}

		return true
	})

	return bareLines
}

// TestTodoImplCloseHandled asserts that every db.Close() call in
// todo_impl.go has its error return explicitly handled (assigned to _).
func (s *GosecComplianceSuite) TestTodoImplCloseHandled() {
	bareLines := s.findBareCloseCalls("todo_impl.go")
	s.Empty(bareLines,
		"todo_impl.go has bare db.Close() calls (G104) on lines %v; "+
			"use '_ = db.Close()' to explicitly discard the error", bareLines)
}

// TestCategoryImplCloseHandled asserts that every db.Close() call in
// category_impl.go has its error return explicitly handled (assigned to _).
func (s *GosecComplianceSuite) TestCategoryImplCloseHandled() {
	bareLines := s.findBareCloseCalls("category_impl.go")
	s.Empty(bareLines,
		"category_impl.go has bare db.Close() calls (G104) on lines %v; "+
			"use '_ = db.Close()' to explicitly discard the error", bareLines)
}

// TestMessageImplCloseHandled asserts that message_impl.go (the reference
// implementation) continues to handle db.Close() correctly. This is a
// regression guard.
func (s *GosecComplianceSuite) TestMessageImplCloseHandled() {
	bareLines := s.findBareCloseCalls("message_impl.go")
	s.Empty(bareLines,
		"message_impl.go has bare db.Close() calls (G104) on lines %v; "+
			"use '_ = db.Close()' to explicitly discard the error", bareLines)
}
