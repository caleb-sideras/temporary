
package temporary

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"calebsideras.com/temporary/src/utils"
	"github.com/a-h/templ"
)
	

func getPartialPageBoostFn(partialPageFn func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer)) func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
	return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
		partialPageFn(w, r, dep, buffer)
		setBoostHeaders(w)
	}
}
	

func getDynamicPageClosure(page PageProps) (func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer), error) {

	pageFn := userFunctionWrapper(page.Handler, page.ParamType)
	if pageFn == nil {
		return nil, errors.New("invalid handlerParams")
	}

	return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
			err := pageFn(w, r, dep).Render(r.Context(), buffer)
			if err != nil {
				//set some error stuff
			}
		},
		nil
}


func getStaticPageClosure(page PageProps) (func(http.ResponseWriter, *http.Request, utils.Config, *bytes.Buffer), error) {

	pageDir := filepath.Clean(filepath.Join(HTML_OUT_DIR, page.Path, PAGE_BODY_OUT_FILE))

	return func(w http.ResponseWriter, r *http.Request, def utils.Config, buffer *bytes.Buffer) {

			pageTpl, err := template.ParseFiles(pageDir)
			if err != nil {
				panic(fmt.Errorf("Error parsing pre-rendered %s from path: %s\n%v", PAGE_BODY_OUT_FILE, pageDir, err))
			}

			pageTpl.Execute(buffer, nil)

		},
		nil
}		

			
func getDynamicRouteClosure(route RouteProps) (func(http.ResponseWriter, *http.Request,utils.Config, *bytes.Buffer), error) {

	routeFn := userFunctionWrapper(route.Handler, route.ParamType)
	if routeFn == nil {
		return nil, errors.New("invalid handlerParams")
	}

	return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
			err := routeFn(w, r, dep).Render(r.Context(), buffer)
			if err != nil {
				//set some error stuff
			}
		},
		nil

}

			
func getStaticRouteClosure(route RouteProps) (func(http.ResponseWriter, *http.Request,utils.Config, *bytes.Buffer), error) {

	pageDir := filepath.Clean(filepath.Join(HTML_OUT_DIR, route.Path, ROUTE_OUT_FILE))

	return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {

			pageTpl, err := template.ParseFiles(pageDir)
			if err != nil {
				panic(fmt.Errorf("Error parsing pre-rendered %s from path: %s\n%v", ROUTE_OUT_FILE, pageDir, err))
			}

			pageTpl.Execute(buffer, nil)

		},
		nil
}


func getStaticFullPageClosure(page PageProps, index IndexProps, indexPath string) (func(http.ResponseWriter, *http.Request, utils.Config, *bytes.Buffer), error) {

	fullPageDir := filepath.Clean(filepath.Join(HTML_OUT_DIR, page.Path, PAGE_OUT_FILE))
	pageDir := filepath.Clean(filepath.Join(HTML_OUT_DIR, page.Path, PAGE_BODY_OUT_FILE))
	indexDir := filepath.Clean(filepath.Join(HTML_OUT_DIR, indexPath, INDEX_OUT_FILE))

	switch index.HandleType {
	case IndexHandle:
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {

			indexTpl, err := template.ParseFiles(indexDir)
			if err != nil {
				panic(fmt.Errorf("Error parsing pre-rendered %s from path: %s\n%v", INDEX_OUT_FILE, indexDir, err))
			}

			_, err = indexTpl.New("page").ParseFiles(pageDir)
			if err != nil {
				panic(fmt.Errorf("Error parsing pre-rendered %s from path: %s\n%v", PAGE_BODY_OUT_FILE, pageDir, err))
			}

			indexTpl.Execute(buffer, nil)

		}, nil

	case IndexRender:
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
			fullPageTpl, err := template.ParseFiles(fullPageDir)
			if err != nil {
				panic(fmt.Errorf("Error parsing pre-rendered %s from path: %s\n%v", PAGE_OUT_FILE, pageDir, err))
			}

			fullPageTpl.Execute(buffer, nil)
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("something"))
}		


func getDynamicFullPageClosure(page PageProps, index IndexProps, indexPath string) (func(http.ResponseWriter, *http.Request,utils.Config, *bytes.Buffer), error) {

	pageFn := userFunctionWrapper(page.Handler, page.ParamType)
	if pageFn == nil {
		return nil, errors.New("invalid handlerParams")
	}

	switch index.HandleType {
	case IndexHandle:
		indexFn := userFunctionWrapper(index.Handler, index.ParamType)
		if indexFn == nil {
			return nil, errors.New("invalid handlerParams")
		}

		return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {
			err := indexFn(w, r, dep).Render(templ.WithChildren(r.Context(), pageFn(w, r, dep)), buffer)
			if err != nil {
				//set some error stuff
			}
		}, nil

	case IndexRender:

		dir := filepath.Clean(filepath.Join(HTML_OUT_DIR, indexPath, INDEX_OUT_FILE))

		return func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer) {

			indexTpl, err := template.ParseFiles(dir)
			if err != nil {
				panic(fmt.Errorf("Error parsing index.html from path: %s\n%v", dir, err))
			}

			pageTpl, err := templ.ToGoHTML(r.Context(), pageFn(w, r, dep))

			if err != nil {
				panic(fmt.Errorf("Error converting page.go output from path: %s to template.HTML\n%v", page.Path, err))
			}

			_, err = indexTpl.New("page").Parse(string(pageTpl))
			indexTpl.Execute(buffer, nil)

			if err != nil {
				panic(fmt.Errorf("Error converting page.go output from path: %s to template.HTML\n%v", page.Path, err))
			}

		}, nil
	}

	return nil, errors.New(fmt.Sprintf("something"))
}


func userFunctionWrapper(fn interface{}, paramType ParamType) func(http.ResponseWriter, *http.Request, utils.Config) templ.Component {
	switch paramType {
	case def:
		setFn := fn.(func() templ.Component)
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config) templ.Component {
			return setFn()
		}
	case dep:
		setFn := fn.(func(utils.Config) templ.Component)
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config) templ.Component {
			return setFn(dep)
		}
	case resReq:
		setFn := fn.(func(http.ResponseWriter, *http.Request) templ.Component)
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config) templ.Component {
			return setFn(w, r)
		}
	case resReqDep:
		setFn := fn.(func(http.ResponseWriter, *http.Request, utils.Config) templ.Component)
		return func(w http.ResponseWriter, r *http.Request, dep utils.Config) templ.Component {
			return setFn(w, r, dep)
		}
	default:
		return nil
	}
}		

			
func executeAppropriateFn(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer, page func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer), boostPage func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer), index func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer), boostIndex func(w http.ResponseWriter, r *http.Request, dep utils.Config, buffer *bytes.Buffer)) {
	requestType := determineRequest(r)
	switch requestType {
	case ErrorRequest:
		// handle Error
	case HxGet_Page:
		page(w, r, dep, buffer)
	case HxBoost_Page:
		boostPage(w, r, dep, buffer)
	case HxGet_Index:
		index(w, r, dep, buffer)
	case HxBoost_Index, NormalRequest:
		boostIndex(w, r, dep, buffer)
	}
}

