package instrument

import (
	"go/ast"
	"go/token"
	"go/types"
)

type OpenTelemetry struct {
	TracerName             string
	ContextName            string
	ErrorStatusDescription string

	hasInserts bool
	hasError   bool
}

func (s *OpenTelemetry) Imports() []*types.Package {
	if !s.hasInserts {
		return nil
	}
	pkgs := []*types.Package{
		types.NewPackage("go.opentelemetry.io/otel", ""),
	}
	if s.hasError {
		pkgs = append(pkgs, types.NewPackage("go.opentelemetry.io/otel/codes", "otelCodes"))
	}
	return pkgs
}

func (s *OpenTelemetry) PrefixStatements(spanName string, hasError bool, errorName string) []ast.Stmt {
	s.hasInserts = true
	if hasError {
		s.hasError = hasError
	}

	stmts := []ast.Stmt{
		&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: s.ContextName}, &ast.Ident{Name: "span"}},
			Rhs: []ast.Expr{s.expFuncSet(s.TracerName, spanName)},
		},
		&ast.DeferStmt{Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "span"}, Sel: &ast.Ident{Name: "End"}},
		}},
	}
	if hasError {
		stmts = append(stmts, &ast.DeferStmt{Call: &ast.CallExpr{Fun: s.exprFuncSetSpanError(errorName)}})
	}
	return stmts
}

func (s *OpenTelemetry) expFuncSet(tracerName, spanName string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: &ast.Ident{Name: "otel"}, Sel: &ast.Ident{Name: "Tracer"}},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"` + tracerName + `"`}},
			},
			Sel: &ast.Ident{Name: "Start"},
		},
		Args: []ast.Expr{&ast.Ident{Name: "ctx"}, &ast.BasicLit{Kind: token.STRING, Value: `"` + spanName + `"`}},
	}
}

func (s *OpenTelemetry) exprFuncSetSpanError(errorName string) ast.Expr {
	return &ast.FuncLit{
		Type: &ast.FuncType{},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{X: &ast.Ident{Name: errorName}, Op: token.NEQ, Y: &ast.Ident{Name: "nil"}},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.ExprStmt{X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "span"}, Sel: &ast.Ident{Name: "SetStatus"}},
						Args: []ast.Expr{
							&ast.SelectorExpr{X: &ast.Ident{Name: "otelCodes"}, Sel: &ast.Ident{Name: "Error"}},
							&ast.BasicLit{Kind: token.STRING, Value: `"` + s.ErrorStatusDescription + `"`},
						},
					}},
					&ast.ExprStmt{X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "span"}, Sel: &ast.Ident{Name: "RecordError"}},
						Args: []ast.Expr{
							&ast.Ident{Name: errorName},
						},
					}},
				}},
			},
		}},
	}
}
