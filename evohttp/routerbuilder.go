package evohttp

import (
	"context"
	"net/http"
	"os"
	"path"
	"strings"
)

type RouterBuilder struct {
	router   *Router
	handlers HandlerChain
	basePath string
}

func NewRouterBuilder() *RouterBuilder {
	b := &RouterBuilder{
		router: NewRouter(),
	}
	return b
}

func (b *RouterBuilder) Use(middleware ...Handler) {
	b.handlers = append(b.handlers, middleware...)
}

func (b *RouterBuilder) UseF(middleware ...HandlerFunc) {
	b.Use(funcToHandlers(middleware)...)
}

func (b *RouterBuilder) SubBuilder(relativePath string, handlers ...Handler) *RouterBuilder {
	child := &RouterBuilder{
		router: b.router,
	}
	child.handlers = joinSlice(b.handlers, handlers)
	child.basePath = path.Join(b.basePath, relativePath)
	return child
}

func (b *RouterBuilder) POST(relativePath string, handlers ...Handler) {
	b.router.addRoute(http.MethodPost, path.Join(b.basePath, relativePath), handlers)
}

func (b *RouterBuilder) POSTF(relativePath string, handlers ...HandlerFunc) {
	b.POST(relativePath, funcToHandlers(handlers)...)
}

func (b *RouterBuilder) GET(relativePath string, handlers ...Handler) {
	b.router.addRoute(http.MethodGet, path.Join(b.basePath, relativePath), handlers)
}

func (b *RouterBuilder) GETF(relativePath string, handlers ...HandlerFunc) {
	b.GET(relativePath, funcToHandlers(handlers)...)
}

func (b *RouterBuilder) HEAD(relativePath string, handlers ...Handler) {
	b.router.addRoute(http.MethodHead, path.Join(b.basePath, relativePath), handlers)
}

func (b *RouterBuilder) HEADF(relativePath string, handlers ...HandlerFunc) {
	b.HEAD(relativePath, funcToHandlers(handlers)...)
}

func (b *RouterBuilder) Static(relativePath string, root string) {
	b.StaticFS(relativePath, Dir(root, false))
}

func (b *RouterBuilder) StaticFS(relativePath string, fs http.FileSystem) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := b.createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// Register GET and HEAD handlers
	b.GET(urlPattern, handler)
	b.HEAD(urlPattern, handler)
}

func (b *RouterBuilder) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(b.basePath, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(c context.Context, req any, info *RequestInfo) (any, error) {

		file := info.UrlParam("filepath")
		// Check if file exists and/or if we have permission to access it
		f, err := fs.Open(file)
		if err != nil {
			info.Writer.WriteHeader(http.StatusNotFound)
			return nil, nil
		}

		if _, noListing := fs.(*onlyFilesFS); noListing {
			stat, err := f.Stat()
			f.Close()
			if err != nil || stat.IsDir() {
				f.Close()
				info.Writer.WriteHeader(http.StatusNotFound)
				return nil, nil
			}
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(&info.Writer, info.Request)
		return nil, nil
	}
}

func funcToHandlers(h []HandlerFunc) []Handler {
	var hs []Handler
	for _, v := range h {
		hs = append(hs, HandlerFunc(v))
	}
	return hs
}

func (b *RouterBuilder) handle(httpMethod, relativePath string, handlers HandlerChain) {
	urlPath := path.Join(b.basePath, relativePath)
	handlers = joinSlice(b.handlers, handlers)
	b.router.addRoute(httpMethod, urlPath, handlers)
}

func joinSlice[T any](a ...[]T) []T {
	var x []T
	for _, v := range a {
		x = append(x, v...)
	}
	return x
}

// Dir returns a http.FileSystem that can be used by http.FileServer(). It is used internally
// in router.Static().
// if listDirectory == true, then it works the same as http.Dir() otherwise it returns
// a filesystem that prevents http.FileServer() to list the directory files.
func Dir(root string, listDirectory bool) http.FileSystem {
	fs := http.Dir(root)
	if listDirectory {
		return fs
	}
	return &onlyFilesFS{fs}
}

type onlyFilesFS struct {
	fs http.FileSystem
}

// Open conforms to http.Filesystem.
func (fs onlyFilesFS) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return neuteredReaddirFile{f}, nil
}

type neuteredReaddirFile struct {
	http.File
}

// Readdir overrides the http.File default implementation.
func (f neuteredReaddirFile) Readdir(_ int) ([]os.FileInfo, error) {
	// this disables directory listing
	return nil, nil
}
