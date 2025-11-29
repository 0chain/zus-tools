package router

import (
	"context"
	"log"
	"net/http"
	"time"
	"zs3server-listener/internal/handler"
)

type Router struct {
	mux              *http.ServeMux
	port             string
	server           *http.Server
	zs3serverHandler handler.ZS3ServerHandlerPort
}

// NewRouter constructs a Router.
func NewRouter(port string, zs3serverHandler handler.ZS3ServerHandlerPort) *Router {
	return &Router{
		mux:              nil,
		port:             port,
		zs3serverHandler: zs3serverHandler,
	}
}

// InitRouter initializes mux and routes.
func (r *Router) InitRouter() {
	r.mux = http.NewServeMux()
	r.initZS3ServerRoutes()
}

// initRoutes registers routes on the mux.
func (r *Router) initZS3ServerRoutes() {

	r.mux.HandleFunc("/zs3server/health", methodHandler(http.MethodGet, r.zs3serverHandler.HealthCheck))

	// enforce method for each route using methodHandler helper
	r.mux.HandleFunc("/increase/ebs", methodHandler(http.MethodGet, r.zs3serverHandler.IncreaseEBS))
}

// Run starts the HTTP server (blocks). It also sets reasonable timeouts.
func (r *Router) Run() error {
	r.server = &http.Server{
		Addr:         ":" + r.port,
		Handler:      r.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("starting server on %s\n", r.server.Addr)
	return r.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server with a context timeout.
func (r *Router) Shutdown(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}

// methodHandler returns an http.HandlerFunc that only allows the given method and
// returns 405 Method Not Allowed for other methods.
func methodHandler(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		h(w, req)
	}
}
