package web

import (
	"fmt"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/auth/provider"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/web"
)

// Run the auth web service
func Run(ctx *cli.Context, opts ...micro.Option) {
	srv := web.NewService(
		web.Name("go.micro.web.auth"),
		web.Version("latest"),
	)

	config := new(map[string]provider.Options)
	err := srv.Options().Service.Options().Config.Get("micro", "auth", "providers").Scan(&config)
	if err != nil {
		logger.Fatalf("Error parsing micro.auth.providers config: %v", err)
	}
	if config == nil {
		logger.Fatalf("Missing required config: micro.auth.providers")
	}

	fmt.Println(config)
}
