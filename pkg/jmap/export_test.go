package jmap

// This is for functions that are only supposed to be visible in tests.

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"
)

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
			for _, syn := range p.Syntax {
				for _, decl := range syn.Decls {
					switch g := decl.(type) {
					case *ast.GenDecl:
						for _, s := range g.Specs {
							switch e := s.(type) {
							case *ast.ValueSpec:
								for i, ident := range e.Names {
									if ident != nil && strings.HasSuffix(ident.Name, suffix) {
										value := e.Values[i]
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
				}
			}
		}
	}
	return result, nil
}
