// Package profiler implements pprof routes.
package profiler

import (
	"net/http"
	"net/http/pprof"
	"path"
	runtime "runtime/pprof"
	"strings"
)

type handler struct {
	mux *http.ServeMux
}

var _ http.Handler = handler{}

func (p handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

var _defaultRoutes = DefaultRoutes()

// DefaultRoutes returns default routes.
//
// Route name is "/debug/pprof/<name>".
func DefaultRoutes() []string {
	// Enable all routes by default except cmdline (unsafe).
	return []string{
		// From pprof.<Name>.
		"profile",
		"symbol",
		"trace",

		// From pprof.Handler(<name>).
		"goroutine",
		"heap",
		"threadcreate",
		"block",
	}
}

// Options for New.
type Options struct {
	Routes       []string           // defaults to DefaultRoutes
	UnknownRoute func(route string) // defaults to ignore
}

// New returns new pprof handler.
func New(opt Options) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/debug/pprof/", pprof.Index)
	routes := opt.Routes
	if len(routes) == 0 {
		routes = _defaultRoutes
	}
	unknown := opt.UnknownRoute
	if unknown == nil {
		unknown = func(route string) {}
	}
	for _, name := range routes {
		name = strings.TrimSpace(name)
		route := path.Join("/debug/pprof/", name)
		switch name {
		case "cmdline":
			m.HandleFunc(route, pprof.Cmdline)
		case "profile":
			m.HandleFunc(route, pprof.Profile)
		case "symbol":
			m.HandleFunc(route, pprof.Symbol)
		case "trace":
			m.HandleFunc(route, pprof.Trace)
		default:
			if runtime.Lookup(name) == nil {
				unknown(name)
				continue
			}
			m.Handle(route, pprof.Handler(name))
		}
	}
	return handler{mux: m}
}
