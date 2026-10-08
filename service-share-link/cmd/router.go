package main

import (
	"net/http"
	"service-share-link/internal/platform/deps"
	"service-share-link/internal/platform/ratelimit"
	sharelink "service-share-link/internal/share-link"
	"service-share-link/internal/share-link/datastore"
	"service-share-link/internal/share-link/service"
)

func NewSeverRouter(cfgDeps *deps.ConfigDeps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	shareLinkService := service.NewService(
		datastore.NewRepoShareLink(cfgDeps.Pool()),
		datastore.NewRedisCache(cfgDeps.RedisClient()),
		datastore.NewRedisViewCounter(cfgDeps.RedisClient()),
	)
	sharelink.RegisterRouterShareLink(mux, shareLinkService, ratelimit.NewLimiter(cfgDeps.RedisClient()))

	return mux
}
