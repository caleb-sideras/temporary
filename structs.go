package temporary

import (
	"net/http"
)

type HandleType int64

const (
	IndexHandle HandleType = iota
	IndexRender
	PageHandle
	PageRender
	RouteHandle
	RouteRender
	FuncError
)

type ParamType int64

const (
	def       ParamType = iota // no params
	resReqDep                  // Response, Request, Dependency
	resReq                     // Response, Request
	dep                        // Dependency
	paramErr
)

type Handler struct {
	Path    string
	Handler interface{}
	// Handler func(w http.ResponseWriter, r *http.Request) templ.Component

	HandleType
}

// so the 3rd type of function will be known at BUILD, so we must create this type then? i.e below
// t := reflect.TypeOf(v) // main.MyStruct
// type ResReqDepHandleFunc struct {
// 	Path    string
// 	Handler func(w http.ResponseWriter, r *http.Request, main.MyStruct ) templ.Component
// }

// Proof of Concept
// temporary := temp.Temp{}
// t := reflect.TypeOf(temporary)
// fmt.Println("Type of v:", t)
// fmt.Println("Package Path:", t.PkgPath())

/**
 * Structs used to organize 'route.go', 'page.go' & 'index.go' handlers for build-files
 **/

type BaseProps struct {
	Path    string
	Handler interface{}
	ParamType
}

type RouteProps struct {
	Path    string
	Handler interface{}
	ParamType
}

type PageProps struct {
	Path    string
	Handler interface{}
	ParamType
	// IndexPath string // TODO: soon will be []string?
}

type IndexProps struct {
	Path    string
	Handler interface{}
	ParamType
	HandleType
}

/**
 * Used for when a HandleFunc is statically rendered but still has w & r params (if these params are used within the func bad stuff will happen)
 **/

type DummyResponseWriter struct{}

func (d DummyResponseWriter) Header() http.Header {
	return http.Header{}
}

func (d DummyResponseWriter) Write(bytes []byte) (int, error) {
	return len(bytes), nil
}

func (d DummyResponseWriter) WriteHeader(statusCode int) {
}

// ok caleb im not sure what you cooked here but it seems we need SEPARATE wrapper handlers to accomdate each of these types. these types will be determind by the AST, not type assertion, so we can do some stupid shit. relax, these gentics look cool tho

// type Default func() templ.Component

// type ResReq func(w http.ResponseWriter, r *http.Request) templ.Component

// type Dep[Dependency any] func(d Dependency) templ.Component

// type ResReqUserDep[Dependency any] func(w http.ResponseWriter, r *http.Request, d Dependency) templ.Component
