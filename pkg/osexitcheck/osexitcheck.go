// Package osexitcheck содержит анализатор запрещающий прямой вызов os.Exit в main функции main пакета
package osexitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OsExitCheckAnalyzer определяет переменную типа analysis.Analyzer для текущего анализатора.
var OsExitCheckAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "prevents direct calls os.Exit in the main function of package main",
	Run:  run,
}

// run реализует работу анализатора.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.File:
				if x.Name.Name != "main" || ast.IsGenerated(x) {
					return false
				}
			case *ast.FuncDecl:
				if x.Name.Name != "main" {
					return false
				}
			case *ast.CallExpr:
				if selector, ok := x.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := selector.X.(*ast.Ident); ok {
						if ident.Name == "os" && selector.Sel.Name == "Exit" {
							pass.Reportf(ident.NamePos, "direct call os.Exit in main package in main function")
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
