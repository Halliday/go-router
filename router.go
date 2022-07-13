package router

import (
	_ "embed"
	"net/http"
	"regexp"
	"strings"

	"github.com/halliday/go-module"
	"github.com/halliday/go-tools"
)

//go:embed messages.csv
var messages string

var _, e, Module = module.New("router", messages)

type Wildcard struct {
	Name    string
	RegExp  *regexp.Regexp
	Handler http.Handler
}

type Route struct {
	Methods  map[string]http.Handler
	Paths    map[string]http.Handler
	Wildcard []*Wildcard
	Next     http.Handler
}

func (r Route) methods() []string {
	methods := make([]string, len(r.Methods))
	i := 0
	for m := range r.Methods {
		methods[i] = m
		i++
	}
	return methods
}

func (r *Route) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/")

	if path == "" {
		var handler http.Handler
		if r.Methods != nil {
			handler = r.Methods[req.Method]
		}
		if handler != nil {
			req.URL.Path = "/"
			handler.ServeHTTP(resp, req)
			return
		}
	} else {

		for _, w := range r.Wildcard {
			if err := req.ParseForm(); err != nil {
				tools.ServeError(resp, err)
				return
			}
			if w.RegExp != nil {
				match := w.RegExp.FindStringSubmatch(path)
				if len(match) == 0 {
					continue
				}
				if len(match) == 1 {
					req.Form.Set(w.Name, match[0])
				} else {
					req.Form[w.Name] = match[1:]
				}
				req.URL.Path = path[len(match[0]):]
				w.Handler.ServeHTTP(resp, req)
				return
			}
			element, subpath := path, "/"
			if i := strings.IndexByte(path, '/'); i != -1 {
				element, subpath = path[:i], path[i:]
			}
			req.URL.Path = subpath
			req.Form.Set(w.Name, element)
			w.Handler.ServeHTTP(resp, req)
			return
		}

		if r.Paths != nil {
			element, subpath := path, "/"
			if i := strings.IndexByte(path, '/'); i != -1 {
				element, subpath = path[:i], path[i:]
			}

			subhandler := r.Paths[element]
			if subhandler != nil {
				req.URL.Path = subpath
				subhandler.ServeHTTP(resp, req)
				return
			}
		}
	}

	if r.Next != nil {
		r.Next.ServeHTTP(resp, req)
		return
	}

	if path == "" && len(r.Methods) != 0 {
		if req.Method == http.MethodOptions {
			requestMethod := req.Header.Get("Access-Control-Request-Method")
			if requestMethod != "" {
				if r.Methods[requestMethod] == nil {
					resp.Header().Set("Access-Control-Allow-Methods", strings.Join(r.methods(), ", "))
					tools.ServeError(resp, e("method_not_allowed"))
					return
				}
			}
			return
		}
		tools.ServeError(resp, e("method_not_allowed"))
		return
	}

	tools.ServeError(resp, e("not_found"))
}
