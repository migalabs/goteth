package utils

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A time.Ticker that is never stopped is never collected: it goes on waking the
// runtime at its interval for the life of the process. That is harmless for a
// ticker created once per long-running routine and a leak for one created
// inside a loop.
//
// goteth had four of the second kind, all in retry or rate-limit paths, so they
// accumulated exactly when the node was already under stress (issue #295). A
// one-shot wait wants time.Sleep; a real ticker wants Stop.
//
// This walks the repository because the rule is about every package, and
// because a convention nobody can see is one that gets broken again.
func TestEveryTickerIsStopped(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	checked := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Vendored and generated trees are not ours to police.
			switch info.Name() {
			case ".git", "vendor", "go-relay-client":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			named, unnamed := tickersIn(fn)
			for name, pos := range named {
				checked++
				if !stopsTicker(fn, name) {
					t.Errorf("%s:%d: ticker %q is never stopped. If it is a one-shot "+
						"wait use time.Sleep; if it is a real ticker add defer %s.Stop(). "+
						"An unstopped ticker keeps firing for the life of the process.",
						rel, fset.Position(pos).Line, name, name)
				}
			}
			// A ticker nobody keeps cannot be stopped by anybody. <-time.NewTicker(d).C
			// reads as a one-shot wait and leaks exactly like the assigned version,
			// which is why it is named rather than merely uncounted.
			for _, pos := range unnamed {
				checked++
				t.Errorf("%s:%d: time.NewTicker result is not held, so it can never be "+
					"stopped. For a one-shot wait use time.Sleep.",
					rel, fset.Position(pos).Line)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// If the walk stops finding tickers, this test has quietly stopped testing
	// anything - a rename of the package or a move of the tree would do it.
	if checked == 0 {
		t.Fatal("found no time.NewTicker calls at all; this test is no longer looking " +
			"at the right tree")
	}
	t.Logf("checked %d tickers", checked)
}

// tickersIn returns the variables in fn assigned from time.NewTicker, and the
// positions of any NewTicker call whose result is not bound to a variable at
// all. The second group needs no Stop check: nothing holds them, so nothing can
// stop them.
func tickersIn(fn *ast.FuncDecl) (map[string]token.Pos, []token.Pos) {
	named := map[string]token.Pos{}
	bound := map[token.Pos]bool{}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		if !isCallTo(assign.Rhs[0], "time", "NewTicker") {
			return true
		}
		bound[assign.Rhs[0].Pos()] = true
		if name, ok := assign.Lhs[0].(*ast.Ident); ok {
			named[name.Name] = assign.Pos()
		}
		return true
	})

	var unnamed []token.Pos
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isCallTo(call, "time", "NewTicker") || bound[call.Pos()] {
			return true
		}
		unnamed = append(unnamed, call.Pos())
		return true
	})
	return named, unnamed
}

// stopsTicker reports whether fn calls name.Stop() anywhere, deferred or not.
func stopsTicker(fn *ast.FuncDecl, name string) bool {
	stopped := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if isCallTo(n, name, "Stop") {
			stopped = true
		}
		return !stopped
	})
	return stopped
}

func isCallTo(n ast.Node, receiver, method string) bool {
	var fun ast.Expr
	switch v := n.(type) {
	case *ast.CallExpr:
		fun = v.Fun
	case *ast.DeferStmt:
		fun = v.Call.Fun
	case *ast.ExprStmt:
		call, ok := v.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		fun = call.Fun
	default:
		return false
	}
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != method {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == receiver
}
