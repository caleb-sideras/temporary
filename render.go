package temporary

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"calebsideras.com/temporary/temporary/utils"
	"github.com/a-h/templ"
)

// Render() renders all static files defined by the user
func (g *Temp) Render() {

	fmt.Println("------------------------RENDERING STATIC FILES-------------------------")

	r, _ := http.NewRequest("GET", "/", nil)
	w := DummyResponseWriter{}

	output := ""
	for path, indexProps := range Index {

		fmt.Println("Directory:", path)
		fmt.Println("   -", INDEX_OUT_FILE)

		fp, err := utils.CreateFile(filepath.Join(path, INDEX_OUT_FILE), HTML_OUT_DIR)
		defer fp.Close()

		if err != nil {
			panic(err)
		}

		templOut, err := g.invokeHandlerFunction(indexProps.ParamType, indexProps.Handler, w, r)
		if err != nil {
			continue
		}

		var buffer bytes.Buffer

		err = templOut.Render(templ.WithChildren(context.Background(), utils.PageTemplate()), &buffer)
		if err != nil {
			panic(err)
		}

		metadata := convertStringListToBytesBuffer(indexProps.Metadata)

		addMetadataIntoBuffer(&buffer, metadata)

		_, err = fp.Write(buffer.Bytes())
		if err != nil {
			panic(err)
		}

		// pathAndTagPage, err := readFileAndGenerateETag(HTML_OUT_DIR, filepath.Join(path, PAGE_OUT_FILE))
		// if err != nil {
		// 	panic(err)
		// }

		// output += pathAndTagPage

	}

	for _, pageProps := range PageStatic {

		fmt.Println("Directory:", pageProps.Path)
		fmt.Println("   -", PAGE_OUT_FILE)

		f, err := utils.CreateFile(filepath.Join(pageProps.Path, PAGE_OUT_FILE), HTML_OUT_DIR)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		indexPath, ok := PathToIndex[pageProps.Path]
		if !ok {
			panic(fmt.Errorf("Could not find an index for path: %s", pageProps.Path))
		}

		indexProps, ok := Index[indexPath]
		if !ok {
			panic(fmt.Errorf("Could not find an index for indexKey: %s derived from path: %s", indexPath, pageProps.Path))
		}

		pageOut, err := g.invokeHandlerFunction(pageProps.ParamType, pageProps.Handler, w, r)
		if err != nil {
			panic(fmt.Errorf("Error invoking page.go func from path: %s", pageProps.Path))
		}

		metadata := convertStringListToBytesBuffer(pageProps.Metadata)

		// page.html
		switch indexProps.HandleType {
		case IndexRender:
			// parse ALREADY rendered index static file
			dir := filepath.Clean(filepath.Join(HTML_OUT_DIR, indexPath, INDEX_OUT_FILE))
			indexTpl, err := template.ParseFiles(dir)
			if err != nil {
				panic(fmt.Errorf("Error parsing index.html from path: %s\n%v", dir, err))
			}

			// convert page templ.Component to template.HTML
			pageTpl, err := templ.ToGoHTML(context.Background(), pageOut)
			if err != nil {
				panic(fmt.Errorf("Error converting page.go output from path: %s to template.HTML\n%v", pageProps.Path, err))
			}

			// parse & execute
			_, err = indexTpl.New("page").Parse(string(pageTpl))

			if err != nil {
				panic(fmt.Errorf("Error converting page.go output from path: %s to template.HTML\n%v", pageProps.Path, err))
			}

			var buffer bytes.Buffer

			err = indexTpl.Execute(&buffer, nil)

			if err != nil {
				panic(fmt.Errorf("Error executing buffer of path: %s\n%v", pageProps.Path, err))
			}

			addMetadataIntoBuffer(&buffer, metadata)

			_, err = f.Write(buffer.Bytes())
			if err != nil {
				panic(err)
			}

		}

		pathAndTagPage, err := readFileAndGenerateETag(HTML_OUT_DIR, filepath.Join(pageProps.Path, PAGE_OUT_FILE))
		if err != nil {
			panic(err)
		}
		output += pathAndTagPage

		// page-body.html
		fmt.Println("   -", PAGE_BODY_OUT_FILE)

		fb, err := utils.CreateFile(filepath.Join(pageProps.Path, PAGE_BODY_OUT_FILE), HTML_OUT_DIR)
		if err != nil {
			panic(err)
		}
		defer fb.Close()

		var buffer bytes.Buffer

		err = pageOut.Render(context.Background(), &buffer)
		if err != nil {
			panic(err)
		}

		_, err = fb.Write(buffer.Bytes())
		if err != nil {
			panic(err)
		}

		// page-body-metadata.html
		fmt.Println("   -", PAGE_BODY_OUT_FILE_W_METADATA)

		fbm, err := utils.CreateFile(filepath.Join(pageProps.Path, PAGE_BODY_OUT_FILE_W_METADATA), HTML_OUT_DIR)
		if err != nil {
			panic(err)
		}
		defer fbm.Close()

		var bufferM bytes.Buffer

		pageMetadata := initPageMetadataVar(pageProps.Metadata)
		bufferM.Write(pageMetadata.Bytes())

		err = pageOut.Render(context.Background(), &bufferM)
		if err != nil {
			panic(err)
		}

		_, err = fbm.Write(bufferM.Bytes())
		if err != nil {
			panic(err)
		}

		pathAndTagBody, err := readFileAndGenerateETag(HTML_OUT_DIR, filepath.Join(pageProps.Path, PAGE_BODY_OUT_FILE_W_METADATA))
		if err != nil {
			panic(err)
		}
		output += pathAndTagBody
	}

	for _, routeProps := range RouteStatic {

		fmt.Println("Directory:", routeProps.Path)
		fmt.Println("   -", ROUTE_OUT_FILE)

		fp, err := utils.CreateFile(filepath.Join(routeProps.Path, ROUTE_OUT_FILE), HTML_OUT_DIR)
		defer fp.Close()

		if err != nil {
			panic(err)
		}

		templOut, err := g.invokeHandlerFunction(routeProps.ParamType, routeProps.Handler, w, r)
		if err != nil {
			continue
		}

		err = templOut.Render(context.Background(), fp)
		if err != nil {
			panic(err)
		}

		pathAndTagBody, err := readFileAndGenerateETag(HTML_OUT_DIR, filepath.Join(routeProps.Path, ROUTE_OUT_FILE))
		if err != nil {
			panic(err)
		}
		output += pathAndTagBody

	}

	file, err := utils.CreateFile(ETAG_FILE, HTML_OUT_DIR)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	_, err = file.Write([]byte(output))
	if err != nil {
		panic(err)
	}

}

// TODO: needs to be generated
func (g Temp) invokeHandlerFunction(params ParamType, fn interface{}, w DummyResponseWriter, r *http.Request) (templ.Component, error) {

	var templOut templ.Component
	switch params {
	case def:
		templOut = fn.(func() templ.Component)()
	case dep:
		templOut = fn.(func(d interface{}) templ.Component)(g.dependency)
	case resReq:
		templOut = fn.(func(w http.ResponseWriter, r *http.Request) templ.Component)(w, r)
	case resReqDep:
		templOut = fn.(func(w http.ResponseWriter, r *http.Request, d interface{}) templ.Component)(w, r, g.dependency)
	case paramErr:
		return nil, errors.New(fmt.Sprintf("something"))
	default:
		return nil, errors.New(fmt.Sprintf("something"))
	}

	return templOut, nil
}

func readFileAndGenerateETag(outDir string, filePath string) (string, error) {

	content, err := os.ReadFile(filepath.Join(outDir, filePath))
	if err != nil {
		return "", err
	}
	output := fmt.Sprintf("%s:%s\n", filePath, utils.GenerateETag(string(content)))
	return output, nil

}
