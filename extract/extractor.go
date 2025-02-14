package extract

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	keyType = "github.com/SLASH2NL/lingua.Key"
)

// KeysFromSource finds all `github.com/SLASH2NL/lingua.Key` used in go source files in dir recusively.
// It will not traverse into imports.
func KeysFromSource(dir string) ([]string, error) {
	dirs, err := findDirsRecursively(dir)
	if err != nil {
		return nil, err
	}

	var translations []string
	for _, dir := range dirs {
		fset := token.NewFileSet()

		mode := packages.NeedName | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedCompiledGoFiles

		cfg := &packages.Config{
			Mode:  mode,
			Dir:   dir,
			Fset:  fset,
			Tests: false,
		}

		pkgs, err := packages.Load(cfg)
		if err != nil {
			return nil, fmt.Errorf("loading package: %w", err)
		}

		pkgsErrs := ""
		packages.Visit(pkgs, nil, func(pkg *packages.Package) {
			for _, err := range pkg.Errors {
				if strings.HasPrefix(err.Msg, "build constraints exclude all Go files") {
					continue
				}

				pkgsErrs += err.Error() + "\n"
			}
		})
		if pkgsErrs != "" {
			return nil, fmt.Errorf("package load error: %s", pkgsErrs)
		}

		for _, pkg := range pkgs {
			for ident, def := range pkg.TypesInfo.Types {
				if def.Type.String() == keyType && def.Value != nil {
					translations = append(translations, strings.Trim(def.Value.ExactString(), "\""))
				} else if callExpr, ok := ident.(*ast.CallExpr); ok {
					keys := processCallExpr(pkg.TypesInfo, callExpr)
					if keys != nil {
						translations = append(translations, keys...)
					}
				}
			}
		}
	}

	return removeDuplicates(translations), nil
}

func processCallExpr(info *types.Info, v *ast.CallExpr) []string {
	// It is a direct call to a function.
	ident, ok := v.Fun.(*ast.Ident)
	if ok {
		return keyFromCallExpr(info, ident, v.Args)
	}

	// It is a call to a method.
	tr, ok := v.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	return keyFromCallExpr(info, tr.Sel, v.Args)
}

// keyFromCallExpr returns the value from the given ast.Ident.
// If no translation can be found it will return an empty string.
// It will only resolve translation keys from consts or simple assignments.
func keyFromCallExpr(info *types.Info, ident *ast.Ident, args []ast.Expr) []string {
	typ := info.TypeOf(ident)
	if typ == nil {
		return nil
	}

	sig, ok := typ.(*types.Signature)
	if !ok {
		return nil
	}

	if len(args) != sig.Params().Len() {
		return nil
	}

	keys := make([]string, 0)

	for i := range sig.Params().Len() {
		if sig.Params().At(i).Type().String() == keyType {
			translation := getValueFromExpr(args[i], info)
			if translation != "" {
				keys = append(keys, translation)
			}
		}
	}

	return keys
}

func getValueFromExpr(expr ast.Expr, info *types.Info) string {
	switch argType := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(argType.Value, "\"")
	case *ast.Ident:
		// Handle the case where the argument is an identifier (e.g., a variable or constant)
		obj := info.ObjectOf(argType)
		if obj == nil {
			return ""
		}

		// Depending on the type of object, try to extract the value
		switch v := obj.(type) {
		case *types.Const:
			// If it's a constant, return the constant value as a string
			return strings.Trim(v.Val().ExactString(), "\"")
		case *types.Var:
			// If it is a variable we try to find the value.
			// Note: Accessing Obj() is deprected, but it's the only way to get the declaration.
			switch decl := argType.Obj.Decl.(type) {
			case *ast.ValueSpec:
				for _, value := range decl.Values {
					// Find the first matching string value.
					parsedValue := getValueFromExpr(value, info)
					if parsedValue != "" {
						return parsedValue
					}
				}
			case *ast.AssignStmt:
				// Only support simple assignments like:
				// var translation = "key"
				// not multiple assignments like:
				// var translation, translation2 = "key", "key2"
				return getValueFromExpr(decl.Rhs[0], info)
			}
		}
	case *ast.CallExpr:
		if len(argType.Args) > 0 {
			for _, arg := range argType.Args {
				translation := getValueFromExpr(arg, info)
				if translation != "" {
					return translation
				}
			}
		}
	}

	return ""
}

// findDirsRecursively finds all directories that contain go files in the given root directory.
func findDirsRecursively(rootDir string) ([]string, error) {
	subdirs := []string{rootDir}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != rootDir {
			hasGoFiles := false

			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
					hasGoFiles = true
					break
				}
			}

			if hasGoFiles {
				subdirs = append(subdirs, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return subdirs, nil
}

func removeDuplicates(input []string) []string {
	// Create a map to track seen elements
	seen := make(map[string]bool)
	// Create a slice to store unique elements
	var result []string

	// Iterate over the input slice
	for _, value := range input {
		// If the value has not been seen before, add it to the result
		if !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}

	return result
}
