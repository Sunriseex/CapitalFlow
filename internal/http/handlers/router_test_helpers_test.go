package handlers

import (
	"net/http"

	"github.com/sunriseex/capitalflow/internal/application"
	"github.com/sunriseex/capitalflow/internal/auth"
)

func newTestRouter(store application.Store, cfg *RouterConfig, tokens ...*auth.TokenService) http.Handler {
	if cfg == nil {
		cfg = &RouterConfig{}
	}
	var tokenService *auth.TokenService
	if len(tokens) > 0 {
		tokenService = tokens[0]
	}
	return newTestRouterWithApplicationConfig(store, cfg, application.Config{TokenService: tokenService})
}

func newTestRouterWithApplicationConfig(store application.Store, cfg *RouterConfig, appCfg application.Config) http.Handler {
	app, err := application.New(store, appCfg)
	if err != nil {
		panic("compose test application: " + err.Error())
	}
	return NewRouter(app, cfg)
}
