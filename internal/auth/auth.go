package auth

import (
	"github.com/micro/go-micro/v2/auth"
	"golang.org/x/oauth2"
)

// SystemRules are the default rules which are applied to the runtime services
var SystemRules = map[string][]*auth.Resource{
	"*": {
		&auth.Resource{Namespace: "*", Type: "*", Name: "*", Endpoint: "*"},
	},
	"": {
		&auth.Resource{Namespace: "*", Type: "service", Name: "go.micro.auth", Endpoint: "Auth.Generate"},
		&auth.Resource{Namespace: "*", Type: "service", Name: "go.micro.auth", Endpoint: "Auth.Token"},
		&auth.Resource{Namespace: "*", Type: "service", Name: "go.micro.auth", Endpoint: "Auth.Inspect"},
		&auth.Resource{Namespace: "*", Type: "service", Name: "go.micro.registry", Endpoint: "Registry.GetService"},
		&auth.Resource{Namespace: "*", Type: "service", Name: "go.micro.registry", Endpoint: "Registry.ListServices"},
	},
}

// OauthProviders is a map containing all the oauth endpoints for supported oauth providers
var OauthProviders = map[string]oauth2.Endpoint{
	"amazon": oauth2.Endpoint{
		AuthURL:  "https://www.amazon.com/ap/oa",
		TokenURL: "https://api.amazon.com/auth/o2/token",
	},
	"facebook": oauth2.Endpoint{
		AuthURL:  "https://www.facebook.com/v3.2/dialog/oauth",
		TokenURL: "https://graph.facebook.com/v3.2/oauth/access_token",
	},
	"github": oauth2.Endpoint{
		AuthURL:  "https://github.com/login/oauth/authorize",
		TokenURL: "https://github.com/login/oauth/access_token",
	},
	"google": oauth2.Endpoint{
		AuthURL:   "https://accounts.google.com/o/oauth2/auth",
		TokenURL:  "https://oauth2.googleapis.com/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
}
