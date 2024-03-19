package temporary

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type tempDir struct {
	FileType string
	FilePath string
}

type fnType struct {
	Recv   string   // Receiver type
	Rtn    string   // Return type
	Params []string // Param types
}

var FILE_CHECK_LIST = map[string]bool{
	INDEX_FILE:   true,
	PAGE_FILE:    true,
	ROUTE_FILE:   true,
	PAGE_JS_FILE: true,
	PAGE_TS_FILE: true,
}

const (
	HTML_SERVE_PATH = "/static/"
	APP_DIR         = "src/app"
	PROJECT_PACKAGE = "calebsideras.com/temporary/"
)

var (
	DEPENDENCY_NAME = ""
)

func (t *Temp) Build() {

	DEPENDENCY_NAME = t.dependencyName
	fmt.Println("t.dependencyName", t.dependencyName, "DEPENDENCY_NAME ", DEPENDENCY_NAME)

	fmt.Println("--------------------------WALKING DIRECTORY--------------------------")
	dirFiles, err := walkDirectoryStructure(APP_DIR)
	if err != nil {
		panic(err)
	}
	printDirectoryStructure(dirFiles)

	fmt.Println("-------------------------EXTRACTING YOUR CODE-------------------------")
	imports, pathToIndex, indexSD, pageStatic, pageDynamic, routeStatic, routeDynamic := getSortedFunctions(dirFiles)

	fmt.Println("-----------------------RENDERING SORTED FUNCTIONS----------------------")
	code, err := renderSortedFunctions(imports, pathToIndex, indexSD, pageStatic, pageDynamic, routeStatic, routeDynamic)
	if err != nil {
		panic(err)
	}
	fmt.Println(code)
}

func walkDirectoryStructure(startDir string) (map[string]map[string][]tempDir, error) {

	result := make(map[string]map[string][]tempDir)

	err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && strings.HasPrefix(info.Name(), "_") && !strings.HasSuffix(info.Name(), "_") {
			return filepath.SkipDir
		}

		if info.IsDir() && path != startDir {
			files := make(map[string][]tempDir)

			filepath.Walk(path, func(innerPath string, innerInfo os.FileInfo, innerErr error) error {

				if innerInfo.IsDir() && strings.HasPrefix(innerInfo.Name(), "_") && !strings.HasSuffix(innerInfo.Name(), "_") {
					return filepath.SkipDir
				}

				ext := filepath.Ext(innerPath)
				if !innerInfo.IsDir() && filepath.Dir(innerPath) == path && FILE_CHECK_LIST[filepath.Base(innerPath)] && filepath.Base(innerPath) != INDEX_FILE {
					if _, exists := files[ext]; !exists {
						files[ext] = []tempDir{}
					}
					files[ext] = append(files[ext], tempDir{filepath.Base(innerPath), innerPath})
				}
				return nil
			})

			currDir := path
			for {
				indexFile := filepath.Join(currDir, INDEX_FILE)
				if _, err := os.Stat(indexFile); !os.IsNotExist(err) {
					if _, ok := files[filepath.Ext(indexFile)]; !ok {
						files[filepath.Ext(indexFile)] = []tempDir{}
					}
					files[filepath.Ext(indexFile)] = append(files[filepath.Ext(indexFile)], tempDir{filepath.Base(indexFile), indexFile})
					break
				}
				currDir = filepath.Dir(currDir)
				if currDir == "." || currDir == "/" {
					return errors.New("MISSING: " + INDEX_FILE)
				}
			}

			result[path] = files
		}
		return nil
	})

	return result, err
}

func printDirectoryStructure(dirFiles map[string]map[string][]tempDir) {
	for k, v := range dirFiles {
		fmt.Println("Directory:", k)
		for ext, files := range v {
			fmt.Println("  ", ext)
			for _, file := range files {
				fmt.Println("   -", file)
			}
		}
	}
}

type sortedFunctionsByFunctionality struct {
	imports            map[string]string
	indexStaticDynamic map[string]string
	pathToIndex        map[string]string
	routeStatic        []string
	routeDynamic       []string
	pageStatic         []string
	pageDynamic        []string
}

type sortedFunctionsByParams struct {
	indexGroup map[string]string
	imports    map[string]string
	resReqDep  []string // Response, Request, Dependency
	resReq     []string // Response, Request
	dep        []string // Dependency
	def        []string // no params
}

type funcConfig struct {
	funcName string
	HandleType
}

func getSortedFunctions(dirFiles map[string]map[string][]tempDir) ([]string, []string, []string, []string, []string, []string, []string) {

	var imports map[string]string = make(map[string]string)
	var indexStatic map[string]string = make(map[string]string)
	var indexDynamic map[string]string = make(map[string]string)
	var pageStatic []string
	var pageDynamic []string
	var routeStatic []string
	var routeDynamic []string

	var sf sortedFunctionsByFunctionality = sortedFunctionsByFunctionality{
		imports,
		indexStatic,
		indexDynamic,
		pageStatic,
		pageDynamic,
		routeStatic,
		routeDynamic,
	}

	for dir, files := range dirFiles {
		if len(files) <= 0 {
			continue
		}

		fmt.Println("Directory:", dir)

		var goFiles []tempDir
		if _, ok := files[GO_EXT]; ok {
			goFiles = files[GO_EXT]
		}

		ndir := dirPostfixSuffixRemoval(dir)
		ndir = camelToHyphen(ndir)

		leafPath := strings.Replace(ndir, APP_DIR, "", 1)
		if leafPath == "" {
			leafPath = "/"
		}

		// prevents unnecessary import
		needImport := false

		for _, gd := range goFiles {
			switch gd.FileType {
			case INDEX_FILE:
				err := sf.setIndexFunction(
					gd,
					leafPath,
					funcConfig{EXPORTED_INDEX_STATIC, IndexRender},
					funcConfig{EXPORTED_INDEX, IndexHandle},
				)
				if err != nil {
					break
				}

			case PAGE_FILE:
				err := sf.setPageFunction(
					gd,
					leafPath,
					&needImport,
					funcConfig{EXPORTED_PAGE_STATIC, PageRender},
					funcConfig{EXPORTED_PAGE, PageHandle},
				)
				if err != nil {
					break
				}

			case ROUTE_FILE:
				err := sf.setRouteFunction(
					gd,
					leafPath,
					&needImport,
					funcConfig{"", RouteRender},
					funcConfig{"", RouteHandle},
				)
				if err != nil {
					break
				}

			}
		}

		if needImport {
			// TODO - fix unecassary import
			sf.imports[fmt.Sprintf(`"%s%s"`, PROJECT_PACKAGE, dir)] = ""
		}
	}

	var importFinal []string
	for key := range sf.imports {
		importFinal = append(importFinal, key)
	}

	var pathToIndexFinal []string
	for path, index := range sf.pathToIndex {
		pathToIndexFinal = append(pathToIndexFinal, fmt.Sprintf(`"%s" : "%s",`, path, index))
	}

	var indexStaticDynamicFinal []string
	for path, index := range sf.indexStaticDynamic {
		indexStaticDynamicFinal = append(indexStaticDynamicFinal, fmt.Sprintf(`"%s" : %s,`, path, index))
	}

	return importFinal, pathToIndexFinal, indexStaticDynamicFinal, sf.pageStatic, sf.pageDynamic, sf.routeStatic, sf.routeDynamic
}

// Gets various types of Route functions - returns soft error
func (sf *sortedFunctionsByFunctionality) setRouteFunction(gd tempDir, leafPath string, needImport *bool, static funcConfig, dynamic funcConfig) error {
	fmt.Println("   route.go")

	expFns, pkName, err := getExportedFuctions(gd.FilePath)
	if err != nil {
		return err
	}

	for expFn, expT := range expFns {
		err := determineFunctionDefinition(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		var fnType HandleType

		expFnPath := camelToHyphen(strings.TrimSuffix(expFn, "_"))

		if strings.HasSuffix(expFn, "_") {
			fnType = static.HandleType
		} else {
			fnType = dynamic.HandleType
		}

		fnParams, err := determineFunctionParams(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		/**
		 * NOTE: '@fnProps' conforms to type '@RouteProps'
		 * type RouteProps struct {
		 *	  Path    string
		 *	  Handler interface{}
		 *	  ParamType
		 * }
		 **/
		fnProps := fmt.Sprintf(`{"%s/%s", %s.%s, %d},`, leafPath, expFnPath, pkName, expFn, fnParams)

		sf.addToSortedFunctions(fnType, fnProps, expFn, "", "")

		*needImport = true
		fmt.Printf("   - Extracted -> func %s\n", expFn)
	}

	return nil
}

// Gets various types of Page functions - returns soft error
func (sf *sortedFunctionsByFunctionality) setPageFunction(gd tempDir, leafPath string, needImport *bool, static funcConfig, dynamic funcConfig) error {
	fmt.Println("   page.go")

	expFns, pkName, err := getExportedFuctions(gd.FilePath)
	if err != nil {
		return err
	}

	for expFn, expT := range expFns {

		err := determineFunctionDefinition(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		fnType, err := determineFunctionType(expFn, static, dynamic)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		fnParams, err := determineFunctionParams(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		/**
		 * NOTE: '@fnProps' conforms to type '@PageProps'
		 * type PageProps struct {
		 *	  Path    string
		 *	  Handler interface{}
		 *	  ParamType
		 * }
		 **/
		fnProps := fmt.Sprintf(`{"%s", %s.%s, %d},`, leafPath, pkName, expFn, fnParams)

		sf.addToSortedFunctions(fnType, fnProps, expFn, "", "")

		*needImport = true
		fmt.Printf("   - Extracted -> func %s\n", expFn)
	}
	return nil
}

func (sf *sortedFunctionsByFunctionality) setIndexFunction(gd tempDir, leafPath string, static funcConfig, dynamic funcConfig) error {
	fmt.Println("   index.go")

	expFns, pkName, err := getExportedFuctions(gd.FilePath)
	if err != nil {
		return err
	}

	for expFn, expT := range expFns {

		err := determineFunctionDefinition(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		fnType, err := determineFunctionType(expFn, static, dynamic)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		fnParams, err := determineFunctionParams(expT)
		if err != nil {
			fmt.Printf("   - func %s -> %s\n", expFn, err)
			continue
		}

		indexPath := fmt.Sprintf(
			strings.Replace(
				strings.Replace(
					dirPostfixSuffixRemoval(gd.FilePath),
					APP_DIR,
					"",
					1,
				),
				INDEX_FILE,
				"",
				1,
			),
		)

		if indexPath == "" {
			indexPath = "/"
		}

		/**
		 * NOTE: '@fnProps' conforms to type '@IndexProps'
		 * type IndexProps struct {
		 *	  Path    string
		 *	  Handler interface{}
		 *	  ParamType
		 *	  HandleType
		 * }
		 **/
		fnProps := fmt.Sprintf(`{"%s", %s.%s, %d, %d}`, indexPath, pkName, expFn, fnParams, fnType)

		sf.addToSortedFunctions(fnType, fnProps, expFn, indexPath, leafPath)

		sf.imports[fmt.Sprintf(`"%s%s"`, PROJECT_PACKAGE, filepath.Dir(gd.FilePath))] = ""

		fmt.Printf("   - Extracted -> func %s\n", expFn)
	}
	return nil
}

func hasDefinitionError(pkName string, expFns map[string]fnType, gd tempDir) error {

	if pkName == "" {
		return errors.New(fmt.Sprintf("   - No defined package name in %s", gd.FilePath))
	}

	if expFns == nil {
		return errors.New(fmt.Sprintf("   - No exported functions in %s", gd.FilePath))
	}

	return nil
}

func (sf *sortedFunctionsByFunctionality) addToSortedFunctions(fnHandle HandleType, fnProps string, expFn string, indexPath string, leafPath string) {
	switch fnHandle {
	case IndexHandle:
	case IndexRender:
		sf.indexStaticDynamic[indexPath] = fnProps
		sf.pathToIndex[leafPath] = indexPath
	case PageHandle:
		sf.pageDynamic = append(sf.pageDynamic, fnProps)
	case PageRender:
		sf.pageStatic = append(sf.pageStatic, fnProps)
	case RouteHandle:
		sf.routeDynamic = append(sf.routeDynamic, fnProps)
	case RouteRender:
		sf.routeStatic = append(sf.routeStatic, fnProps)
	case FuncError:
		fmt.Printf("   - Unsupported -> func %s\n", expFn)
	}
}

func determineFunctionType(expFn string, static funcConfig, dynamic funcConfig) (HandleType, error) {

	var handleType HandleType

	switch expFn {
	case static.funcName:
		handleType = static.HandleType
	case dynamic.funcName:
		handleType = dynamic.HandleType
	default:
		return FuncError, errors.New(fmt.Sprintf("Unsupported Function Type -> %s", expFn))
	}

	return handleType, nil
}

func determineFunctionParams(expT fnType) (ParamType, error) {
	var param ParamType
	fmt.Sprintf("DEPENDENCY_NAME", DEPENDENCY_NAME)
	if expT.Params == nil || len(expT.Params) == 0 {
		param = def
	} else if len(expT.Params) == 1 && expT.Params[0] == DEPENDENCY_NAME {
		param = dep
	} else if len(expT.Params) == 2 && expT.Params[0] == "http.ResponseWriter" && expT.Params[1] == "*http.Request" {
		param = resReq
	} else if len(expT.Params) == 3 && expT.Params[0] == "http.ResponseWriter" && expT.Params[1] == "*http.Request" && expT.Params[2] == DEPENDENCY_NAME {
		param = resReqDep
	} else {
		return paramErr, errors.New(fmt.Sprintf("Unsupported Function Params -> %s", expT.Params))
	}
	return param, nil
}

func determineFunctionDefinition(expT fnType) error {

	if expT.Rtn != "templ.Component" {
		return errors.New(fmt.Sprintf("Unsupported Return Type -> %s", expT.Rtn))
	}

	if expT.Recv != "" {
		return errors.New(fmt.Sprintf("Unsupported Receiver Type -> %s", expT.Recv))
	}

	return nil
}

func getExportedFuctions(path string) (map[string]fnType, string, error) {

	node, err := getAstVals(path)
	if err != nil {
		return nil, "", errors.New(fmt.Sprintf("   - Error parsing file: %s\n%s", path, err))
	}

	var pkName string
	expFns := make(map[string]fnType)

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.File:
			pkName = x.Name.Name
		case *ast.FuncDecl:
			if !x.Name.IsExported() {
				break
			}

			fnType := fnType{}

			// Return Type
			if x.Type.Results != nil {
				for _, res := range x.Type.Results.List {
					switch t := res.Type.(type) {
					case *ast.Ident:
						fnType.Rtn = t.Name
					case *ast.SelectorExpr:
						fnType.Rtn = fmt.Sprintf("%s.%s", t.X, t.Sel)
					case *ast.StarExpr:
						if ident, ok := t.X.(*ast.Ident); ok {
							fnType.Rtn = fmt.Sprintf("*%s", ident.Name)
						}
					}
				}
			}

			// Params
			if x.Type.Params != nil {
				for _, param := range x.Type.Params.List {
					for _ = range param.Names {
						exprDetails := ExtractExprDetails(param.Type)
						fnType.Params = append(fnType.Params, formatParams(exprDetails))
					}
				}
			}

			// Receiver Type
			if x.Recv != nil {
				for _, res := range x.Recv.List {
					switch t := res.Type.(type) {
					case *ast.Ident:
						fnType.Recv = t.Name
					case *ast.SelectorExpr:
						fnType.Recv = fmt.Sprintf("%s.%s", t.X, t.Sel)
					case *ast.StarExpr:
						if ident, ok := t.X.(*ast.Ident); ok {
							fnType.Recv = fmt.Sprintf("*%s", ident.Name)
						}
					}
				}
			}
			expFns[x.Name.Name] = fnType

		}
		return true
	})

	if expFns == nil {
		return nil, "", errors.New(fmt.Sprintf("   - No defined package name in %s", path))
	}

	if pkName == "" {
		return nil, "", errors.New(fmt.Sprintf("   - No exported functions in %s", path))
	}

	return expFns, pkName, nil
}

func renderSortedFunctions(imports []string, pathToIndex []string, indexSD []string, pageStatic []string, pageDynamic []string, routeStatic []string, routeDynamic []string) (string, error) {

	code := `
// Code generated by Temporary; DO NOT EDIT.
package temporary
import (
	` + strings.Join(imports, "\n\t") + `
)

var PathToIndex = map[string]string{
	` + strings.Join(pathToIndex, "\n\t") + `
}

var Index = map[string]IndexProps{
	` + strings.Join(indexSD, "\n\t") + `
}

var PageStatic = []PageProps{
	` + strings.Join(pageStatic, "\n\t") + `
}

var PageDynamic = []PageProps{
	` + strings.Join(pageDynamic, "\n\t") + `
}

var RouteStatic = []RouteProps{
	` + strings.Join(routeStatic, "\n\t") + `
}

var RouteDynamic = []RouteProps{
	` + strings.Join(routeDynamic, "\n\t") + `
}
`
	err := os.WriteFile("./temporary/definitions.go", []byte(code), 0644)
	if err != nil {
		return "", err
	}
	return code, nil
}

func getAstVals(path string) (*ast.File, error) {
	_, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func isHTTPResponseWriter(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Check if the type is "http" package and "ResponseWriter" identifier
	return x.Name == "http" && selector.Sel.Name == "ResponseWriter"
}

func formatParams(exprDetails ExprDetails) string {
	var param string
	if exprDetails.IsPointer {
		param = "*" + exprDetails.PackageName + "." + exprDetails.Selector
	} else {
		param = exprDetails.PackageName + "." + exprDetails.Selector
	}

	return param
}

func isHTTPRequest(expr ast.Expr) bool {
	// Check for *http.Request
	starExpr, ok := expr.(*ast.StarExpr)
	if ok {
		expr = starExpr.X
	}

	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Check if the type is "http" package and "Request" identifier
	return x.Name == "http" && selector.Sel.Name == "Request"
}

func ExtractExprDetails(expr ast.Expr) ExprDetails {
	var details ExprDetails

	switch e := expr.(type) {
	// since all paramTypes will be defined in separate packages, we don't really need this
	case *ast.Ident:
		details.Name = e.Name
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			details.PackageName = ident.Name
		}
		details.Name = e.Sel.Name
		details.Selector = e.Sel.Name
	case *ast.StarExpr:
		details.IsPointer = true
		// Extract details from the pointed type.
		pointedDetails := ExtractExprDetails(e.X)
		// Merge details from pointed type.
		details.Name = pointedDetails.Name
		details.Selector = pointedDetails.Selector
		details.PackageName = pointedDetails.PackageName
	}

	return details
}

// ExprDetails captures the extracted details from an ast.Expr.
type ExprDetails struct {
	Name        string // The identifier name or the selector's identifier.
	Selector    string // The name of the selector, if applicable.
	IsPointer   bool   // True if the expression is a pointer.
	PackageName string // The package name, if applicable.
}

func dirPostfixSuffixRemoval(path string) string {
	segments := strings.Split(path, "/")
	var output []string
	if len(segments) == 0 {
		return path
	}
	for _, segment := range segments {
		if strings.HasPrefix(segment, "_") && strings.HasSuffix(segment, "_") {
			s1 := segment[1 : len(segment)-1]
			output = append(output, fmt.Sprintf("{%s}", s1))
		} else if !strings.HasSuffix(segment, "_") {
			output = append(output, segment)
		}
	}
	/**
	* TODO - slugs
	* So we want to take the file path -> _example_ and add it to the filepath as "/{example}"
	* Not sure the effects of this yet in current structure
	* NOTE
	* Have to use name of folder cuz you can access this from request handler -> slug := mux.Vars(r)["example"]
	* ISSUE
	* So it seems the `templ generate` command ignores any dirs with "_" prefix. So templs in slug dirs will be ignored?
	* Can specify dirs -> templ generate -f /home/caleb/go/personal/src/app/_test/test.templ
	**/
	return filepath.Join(output...)
}

func camelToHyphen(input string) string {
	var result bytes.Buffer

	for i, char := range input {
		if i > 0 && unicode.IsUpper(char) {
			result.WriteRune('-')
		}
		result.WriteRune(unicode.ToLower(char))
	}

	return result.String()
}
