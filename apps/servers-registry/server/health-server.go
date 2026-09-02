package server

import (
	"context"
	"net/http"
)

type ReadinessFunc func(ctx context.Context) (bool, error)

func NewHealthServer(port string, ready ReadinessFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		ok, err := ready(r.Context())
		if err != nil || !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"NOT_READY"}`))
			return
		}
		w.Write([]byte(`{"status":"OK"}`))
	})

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
