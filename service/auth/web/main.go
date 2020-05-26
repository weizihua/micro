package web

import (
	"encoding/json"
	"fmt"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/auth/provider"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/web"
	"github.com/micro/micro/v2/service/auth/web/handler/oauth"
)

var providers map[string]*provider.Options

// Run the auth web service
func Run(ctx *cli.Context, opts ...micro.Option) {
	srv := web.NewService(
		web.Name("go.micro.web.auth"),
		web.Version("latest"),
	)

	data := make(map[string]string)
	err := srv.Options().Service.Options().Config.Get("micro", "auth", "providers").Scan(&data)
	if err != nil {
		logger.Fatalf("Error parsing micro.auth.providers config: %v", err)
	}
	if data == nil {
		logger.Fatalf("Missing required config: micro.auth.providers")
	}

	// todo: improve the way we set in the store so we can scan directly into the providers struct
	providers = make(map[string]*provider.Options, len(data))
	for k, v := range data {
		var opts *provider.Options
		json.Unmarshal([]byte(v), &opts)
		providers[k] = opts
	}

	// auth and store are used by oauth providers. auth is used to issue accounts once a users identity
	// has been verified and the store is used to cache oauth states.
	auth := srv.Options().Service.Options().Auth
	store := srv.Options().Service.Options().Store

	// setup the handlers for the oauth providers
	for name, opts := range providers {
		if opts.Type != "oauth" {
			continue
		}

		logger.Infof("Registering oauth/%v handlers", name)

		h := oauth.NewHandler(name, opts, auth, store)
		srv.HandleFunc(fmt.Sprintf("/%v/login", name), h.Login)
		srv.HandleFunc(fmt.Sprintf("/%v/verify", name), h.Verify)
	}

	// todo: if basic auth is enabled register the handler which creates an account

	// todo: register the index handler which will render the providers list to the user

	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}
}
