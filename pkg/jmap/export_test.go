package jmap

// This is for functions that are only supposed to be visible in tests.

import (
	"fmt"
	"go/ast"
	"go/token"
	"iter"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"
)

func valuesOf(p *packages.Package) iter.Seq[*ast.ValueSpec] { //NOSONAR
	return func(yield func(*ast.ValueSpec) bool) {
		for _, syn := range p.Syntax {
			for _, decl := range syn.Decls {
				g, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, s := range g.Specs {
					e, ok := s.(*ast.ValueSpec)
					if !ok {
						continue
					}
					if !yield(e) {
						return
					}
				}
			}
		}
	}
}

func parseConsts(pkgID string, suffix string, typeName string) (map[string]string, error) { //NOSONAR
	result := map[string]string{}
	{
		cfg := &packages.Config{
			Mode:  packages.LoadSyntax,
			Dir:   ".",
			Tests: false,
		}
		pkgs, err := packages.Load(cfg, ".")
		if err != nil {
			log.Fatal(err)
		}
		if packages.PrintErrors(pkgs) > 0 {
			return nil, fmt.Errorf("failed to parse the package '%s'", pkgID)
		}
		for _, p := range pkgs {
			if p.ID != pkgID {
				continue
			}
			for v := range valuesOf(p) {
				for i, ident := range v.Names {
					if ident != nil && strings.HasSuffix(ident.Name, suffix) {
						value := v.Values[i]
						switch c := value.(type) {
						case *ast.CallExpr:
							switch f := c.Fun.(type) {
							case *ast.Ident:
								if f.Name == typeName {
									switch a := c.Args[0].(type) {
									case *ast.BasicLit:
										if a.Kind == token.STRING {
											result[ident.Name] = strings.Trim(a.Value, `"`)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return result, nil
}
