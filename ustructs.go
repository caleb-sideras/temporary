
// Code generated by Temporary; DO NOT EDIT.
// Build-time types for the handlers containing user defined types  

package temporary
import (
	"net/http"
	"github.com/a-h/templ"
)

type ResReqDepHandleFunc struct {
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request, dep interface{}) templ.Component
}

type DepHandleFunc struct {
	Path    string
	Handler func(dep interface{}) templ.Component
}
