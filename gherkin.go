package gherkin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var indentReplacer = strings.NewReplacer("\t", "")

// Scenario accepts a
func Scenario(s string, a ...interface{}) string {
	return strings.TrimSpace(indentReplacer.Replace(fmt.Sprintf(s, a...)))
}

// Results structures a list of files walked during Extract and the
// Scenarios gathered while visiting those files.
type Results struct {
	Walked    []string
	Scenarios []string
}

// Extract visits all the Go source files recursively within the base directory
// provided and returns a list of the files visited and the Gherkin scenarios
// collected.
func Extract(base string) (Results, error) {
	files, err := walkGoFiles(base)
	if err != nil {
		return Results{}, err
	}
	if len(files) == 0 {
		return Results{}, fmt.Errorf("no go source files found within %s", base)
	}
	r := Results{}
	for _, file := range files {
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, file, nil, parser.ParseComments)
		if err != nil {
			return Results{}, err
		}
		v := newVisitor(f)
		ast.Walk(v, f)
		r.Scenarios = append(r.Scenarios, v.gherkins...)
	}
	return r, nil
}

func walkGoFiles(p string) ([]string, error) {
	var files []string
	err := filepath.Walk(p,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".go" {
				files = append(files, path)
			}
			return nil
		})
	return files, err
}

type visitor struct {
	pkgDecl  map[*ast.GenDecl]bool
	gherkins []string
}

func newVisitor(f *ast.File) *visitor {
	decls := make(map[*ast.GenDecl]bool)
	for _, decl := range f.Decls {
		if v, ok := decl.(*ast.GenDecl); ok {
			decls[v] = true
			continue
		}
	}

	return &visitor{
		pkgDecl:  decls,
		gherkins: []string{},
	}
}

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	if ok, scenario := v.extractScenario(n); ok {
		v.gherkins = append(v.gherkins, scenario)
	}

	return v
}

//(*ast.CallExpr)(0xc00009e400)({
//Fun: (*ast.SelectorExpr)(0xc0000ac270)({
//X: (*ast.Ident)(0xc0000a2740)(gherkin),
//Sel: (*ast.Ident)(0xc0000a2760)(Scenario)
//}),
//Lparen: (token.Pos) 1525,
//Args: ([]ast.Expr) (len=1 cap=1) {
//(*ast.BasicLit)(0xc0000a2780)({
//ValuePos: (token.Pos) 1526,
//Kind: (token.Token) STRING,
//Value: (string) (len=206) "`\n\t\t\tScenario: \n\n\t\t\tGiven Bruce has registered a user account\n\t\t\t  And he wants to read other user accounts\n\t\t\t  And he misuses the API or provide bad parameters\n\t\t\t Then he is unable to read user accounts`"
//})
//},
//Ellipsis: (token.Pos) 0,
//Rparen: (token.Pos) 1736
//})
func (v visitor) extractScenario(n ast.Node) (bool, string) {
	ce, ok := n.(*ast.CallExpr)
	if !ok {
		return false, ""
	}
	se, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return false, ""
	}
	if se.Sel.Name != "Scenario" {
		return false, ""
	}
	if ident, ok := se.X.(*ast.Ident); ok {
		if ident.Name != "gherkin" {
			return false, ""
		}
	} else {
		return false, ""
	}
	if len(ce.Args) != 1 {
		return false, ""
	}
	if args, ok := ce.Args[0].(*ast.BasicLit); ok {
		return true, strings.TrimSpace(args.Value)
	}
	return false, ""
}
