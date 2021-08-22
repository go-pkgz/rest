package rest

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileServer provides http.FileServer handler to serve static files from a http.FileSystem,
// prevents directory listing by default and supports spa-friendly mode (off by default) returning /index.html on 404.
// - public defines base path of the url, i.e. for http://example.com/static/* it should be /static
// - local for the local path to the root of the served directory
// - notFound is the reader for the custom 404 html, can be nil for default
type FileServer struct {
	public, root  string
	notFound      io.Reader
	isSpa         bool
	enableListing bool
	handler       http.HandlerFunc
}

// NewFileServer creates file server with optional spa mode and optional direcroty listing (disabled by default)
func NewFileServer(public, local string, options ...FSOpt) (*FileServer, error) {
	res := FileServer{
		public:        public,
		notFound:      nil,
		isSpa:         false,
		enableListing: false,
	}

	root, err := filepath.Abs(local)
	if err != nil {
		return nil, fmt.Errorf("can't get absolute path for %s: %w", local, err)
	}
	res.root = root

	if _, err = os.Stat(root); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path %s doesn't exist: %w", root, err)
	}

	for _, opt := range options {
		err = opt(&res)
		if err != nil {
			return nil, err
		}
	}

	cfs := customFS{
		fs:      http.Dir(root),
		spa:     res.isSpa,
		listing: res.enableListing,
	}
	f := http.StripPrefix(public, http.FileServer(cfs))

	res.handler = func(w http.ResponseWriter, r *http.Request) {
		f.ServeHTTP(w, r)
	}

	if !res.enableListing {
		h, err := custom404Handler(f, res.notFound)
		if err != nil {
			return nil, err
		}
		res.handler = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}

	return &res, nil
}

// ServeHTTP makes FileServer compatible with http.Handler interface
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fs.handler(w, r)
}

// FSOpt defines functional option type
type FSOpt func(fs *FileServer) error

// FsOptSPA turns on SPA mode returning "/index.html" on not-found
func FsOptSPA(fs *FileServer) error {
	fs.isSpa = true
	return nil
}

// FsOptListing turns on directory listing
func FsOptListing(fs *FileServer) error {
	fs.enableListing = true
	return nil
}

// FsOptCustom404 sets custom 404 reader
func FsOptCustom404(fr io.Reader) FSOpt {
	return func(fs *FileServer) error {
		fs.notFound = fr
		return nil
	}
}

// customFS wraps http.FileSystem with spa and no-listing optional support
type customFS struct {
	fs      http.FileSystem
	spa     bool
	listing bool
}

// Open file on FS, for directory enforce index.html and fail on a missing index
func (cfs customFS) Open(name string) (http.File, error) {

	f, err := cfs.fs.Open(name)
	if err != nil {
		if cfs.spa {
			return cfs.fs.Open("/index.html")
		}
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if s.IsDir() && !cfs.listing {
		index := strings.TrimSuffix(name, "/") + "/index.html"
		if _, err := cfs.fs.Open(index); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// respWriter404 intercept Write to provide custom 404 response
type respWriter404 struct {
	http.ResponseWriter
	status int
	msg    []byte
}

func (w *respWriter404) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *respWriter404) Write(p []byte) (n int, err error) {
	if w.status != http.StatusNotFound || w.msg == nil {
		return w.ResponseWriter.Write(p)
	}
	_, err = w.ResponseWriter.Write(w.msg)
	return len(p), err
}

func custom404Handler(next http.Handler, notFound io.Reader) (http.Handler, error) {
	if notFound == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) }), nil
	}

	body, err := ioutil.ReadAll(notFound)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&respWriter404{ResponseWriter: w, msg: body}, r)
	}), nil
}
