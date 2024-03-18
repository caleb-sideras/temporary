package temporary

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"calebsideras.com/temporary/temporary/utils"
	"github.com/gorilla/mux"
)

type requestType int64

const (
	NormalRequest requestType = iota
	HxGet_Index
	HxGet_Page
	HxBoost_Page
	HxBoost_Index
	ErrorRequest
)

type pageHandler func(w http.ResponseWriter, r *http.Request)

func (t *Temp) Run(r *mux.Router, port string) {
	fmt.Println("----------------------------CREATING HANDLERS----------------------------")
	http.Handle("/", r)
	t.handleRoutes(r, t.getETags())
	log.Fatal(http.ListenAndServe(port, nil))
}

func (t *Temp) handleRoutes(r *mux.Router, eTags map[string]string) {
	fmt.Println("Function Type: Page - Static")
	t.setPageStatic(r, eTags)
	fmt.Println("Function Type: Page - Dynamic")
	t.setPageDynamic(r, eTags)
	fmt.Println("Function Type: Route - Static")
	t.setRouteStatic(r, eTags)
	fmt.Println("Function Type: Route - Dynamic")
	t.setRouteDynamic(r, eTags)
}

func (t *Temp) setPageStatic(r *mux.Router, eTags map[string]string) {
	for _, pageProps := range PageStatic {
		// loop variable capture? can be removed if updated to go 1.23?
		currRoute := pageProps.Path
		fmt.Printf("   - %s\n", currRoute)

		indexProps, err := getIndexPropsFromPage(pageProps)
		if err != nil {
			panic(err)
		}

		r.HandleFunc(currRoute+"{slash:/?}", t.setStaticPageHandler(pageProps, indexProps, eTags))
	}
}

func (t *Temp) setPageDynamic(r *mux.Router, eTags map[string]string) {
	for _, pageProps := range PageDynamic {
		// loop variable capture? can be removed if updated to go 1.23?
		currRoute := pageProps.Path
		fmt.Printf("   - %s\n", currRoute)

		indexProps, err := getIndexPropsFromPage(pageProps)
		if err != nil {
			panic(err)
		}

		r.HandleFunc(currRoute+"{slash:/?}", t.setDynamicPageHandler(pageProps, indexProps, eTags))
	}
}

func (t *Temp) setRouteDynamic(r *mux.Router, eTags map[string]string) {
	for _, routeProps := range RouteDynamic {
		currRoute := routeProps
		fmt.Printf("   - %s\n", currRoute.Path)

		r.HandleFunc(currRoute.Path+"{slash:/?}", t.setDynamicRouteHandler(routeProps, eTags))
	}

}

func (t *Temp) setRouteStatic(r *mux.Router, eTags map[string]string) {
	for _, routeProps := range RouteStatic {
		currRoute := routeProps
		fmt.Printf("   - %s\n", currRoute.Path)

		r.HandleFunc(currRoute.Path+"{slash:/?}", t.setStaticRouteHandler(routeProps, eTags))
	}
}

func (g Temp) setDynamicPageHandler(page PageProps, index IndexProps, eTags map[string]string) http.HandlerFunc {

	fullPageFn, err := getDynamicFullPageClosure(page, index, index.Path)
	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", page.Path, err))
	}

	partialPageFn, err := getDynamicPageClosure(page)
	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", page.Path, err))
	}

	partialPageBoostFn := getPartialPageBoostFn(partialPageFn)

	return func(w http.ResponseWriter, r *http.Request) {
		logs := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		var buffer bytes.Buffer

		executeAppropriateFn(w, r, g.dependency, &buffer, partialPageFn, partialPageBoostFn, fullPageFn, fullPageFn)

		eTag := utils.GenerateETag(buffer.String())
		writeRequest(w, r, eTag, buffer.Bytes(), eTags, logs)
	}
}

func (g Temp) setStaticPageHandler(page PageProps, index IndexProps, eTags map[string]string) http.HandlerFunc {

	fullPageFn, err := getStaticFullPageClosure(page, index, index.Path)
	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", page.Path, err))
	}

	partialPageFn, err := getStaticPageClosure(page)
	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", page.Path, err))
	}

	partialPageBoostFn := getPartialPageBoostFn(partialPageFn)

	return func(w http.ResponseWriter, r *http.Request) {
		logs := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		var buffer bytes.Buffer

		executeAppropriateFn(w, r, g.dependency, &buffer, partialPageFn, partialPageBoostFn, fullPageFn, fullPageFn)

		eTag := utils.GenerateETag(buffer.String())
		writeRequest(w, r, eTag, buffer.Bytes(), eTags, logs)
	}

}

func (g Temp) setDynamicRouteHandler(routeProps RouteProps, eTags map[string]string) http.HandlerFunc {

	routeFn, err := getDynamicRouteClosure(routeProps)

	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", routeProps.Path, err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logs := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		var buffer bytes.Buffer

		routeFn(w, r, g.dependency, &buffer)

		eTag := utils.GenerateETag(buffer.String())
		writeRequest(w, r, eTag, buffer.Bytes(), eTags, logs)
	}

}

func (g Temp) setStaticRouteHandler(routeProps RouteProps, eTags map[string]string) http.HandlerFunc {

	routeFn, err := getStaticRouteClosure(routeProps)

	if err != nil {
		panic(fmt.Errorf("Error creating handler for route %s\n%w", routeProps.Path, err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logs := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		var buffer bytes.Buffer

		routeFn(w, r, g.dependency, &buffer)

		eTag := utils.GenerateETag(buffer.String())
		writeRequest(w, r, eTag, buffer.Bytes(), eTags, logs)
	}

}

func getIndexPropsFromPage(pageProps PageProps) (IndexProps, error) {

	indexPath, ok := PathToIndex[pageProps.Path]
	if !ok {
		return IndexProps{}, fmt.Errorf("Could not find an index of path: %s", pageProps.Path)
	}

	indexProps, ok := Index[indexPath]
	if !ok {
		return IndexProps{}, fmt.Errorf("Could not find an index of path %s derived from page path: %s", indexPath, pageProps.Path)
	}

	return indexProps, nil
}

func (t *Temp) getETags() map[string]string {
	eTags := make(map[string]string)

	file, err := os.Open(filepath.Join(HTML_OUT_DIR, ETAG_FILE))
	if err != nil {
		log.Fatalf("Could not create file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) == 2 {
			eTags[parts[0]] = parts[1]
		}
	}
	return eTags
}

func formatRequest(r *http.Request, ifPage func(), ifBPage func(), ifIndex func(), ifBIndex func()) {
	requestType := determineRequest(r)
	switch requestType {
	case ErrorRequest:
		// handle Error
	case HxGet_Page:
		ifPage()
	case HxBoost_Page:
		ifBPage()
	case HxGet_Index:
		ifIndex()
	case HxBoost_Index, NormalRequest:
		ifBIndex()
	}
}

func determineRequest(r *http.Request) requestType {
	if !utils.IsHtmxRequest(r) {
		return NormalRequest
	}

	if !utils.IsHxBoosted(r) {
		if r.URL.Query().Get("index") == "true" {
			return HxGet_Index
		}
		return HxGet_Page
	}

	htmxUrl, err := utils.LastElementOfURL(utils.GetHtmxRequestURL(r))
	if err != nil {
		return ErrorRequest
	}

	if _, ok := PathToIndex[htmxUrl]; !ok {
		return HxBoost_Index
	}

	if PathToIndex[htmxUrl] == PathToIndex[r.URL.Path] {
		return HxBoost_Page
	}

	return HxBoost_Index
}

func setBoostHeaders(w http.ResponseWriter) {
	w.Header().Set("HX-Retarget", "global main")
	w.Header().Set("HX-Reswap", "innerHTML transition:true")
}

func setPageHeaders(w http.ResponseWriter, eTagPath string, eTags map[string]string) {
	w.Header().Set("Vary", "HX-Request")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", eTags[eTagPath])
}

func setRouteRenderHeaders(w http.ResponseWriter, eTagPath string, eTags map[string]string) {
	w.Header().Set("Vary", "HX-Request")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", eTags[eTagPath])
}

func setRouteHeaders(w http.ResponseWriter) {
	w.Header().Set("Vary", "HX-Request")
	w.Header().Set("Cache-Control", "no-cache")
}

func setHeaders(w http.ResponseWriter, eTag string) {
	w.Header().Set("Vary", "HX-Request")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", eTag)
}

func (t *Temp) handleRenderError(err error, w http.ResponseWriter, logs string) {
	if err != nil {
		log.Println(fmt.Sprintf("%s %d", logs, http.StatusInternalServerError))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeRequest(w http.ResponseWriter, r *http.Request, eTag string, content []byte, eTags map[string]string, logs string) {
	if rEtag := r.Header.Get("If-None-Match"); rEtag == eTag {
		log.Println(fmt.Sprintf("%s %d", logs, http.StatusNotModified))
		w.WriteHeader(http.StatusNotModified)
		return
	}
	log.Println(fmt.Sprintf("%s %d", logs, http.StatusOK))
	setHeaders(w, eTag)
	w.Write(content)
}
