package temporary

// func NewTemp(dep interface{}) *Temp {
// 	DetermineType(dep)
// 	return &Temp{dep}
// }

const (
	DIR      = "/"
	GO_EXT   = ".go"
	JS_EXT   = ".js"
	TS_EXT   = ".ts"
	HTML_EXT = ".html"
	TXT_EXT  = ".txt"

	EXPORTED_HANDLE       = "Handle"
	EXPORTED_RENDER       = "Render"
	EXPORTED_INDEX        = "Index"
	EXPORTED_INDEX_STATIC = "Index_"
	EXPORTED_PAGE         = "Page"
	EXPORTED_PAGE_STATIC  = "Page_"

	PAGE     = "page"
	INDEX    = "index"
	ROUTE    = "route"
	ETAG     = "etag_file"
	BODY     = "-body"
	MET_TAG  = "-metadata"
	METADATA = "Metadata"

	PAGE_BODY                     = PAGE + BODY
	PAGE_FILE                     = PAGE + GO_EXT
	PAGE_JS_FILE                  = PAGE + JS_EXT
	PAGE_TS_FILE                  = PAGE + TS_EXT
	INDEX_FILE                    = INDEX + GO_EXT
	ROUTE_FILE                    = ROUTE + GO_EXT
	INDEX_OUT_FILE                = INDEX + HTML_EXT
	PAGE_OUT_FILE                 = PAGE + HTML_EXT
	PAGE_BODY_OUT_FILE            = PAGE_BODY + HTML_EXT
	PAGE_BODY_OUT_FILE_W_METADATA = PAGE_BODY + MET_TAG + HTML_EXT
	ROUTE_OUT_FILE                = ROUTE + HTML_EXT
	ETAG_FILE                     = ETAG + TXT_EXT

	HTML_OUT_DIR = "./static/html/"
)
