package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/auth/provider"
	configPb "github.com/micro/go-micro/v2/config/source/service/proto"
	"github.com/micro/micro/v2/internal/auth"
	"github.com/micro/micro/v2/internal/client"
)

func listProviders(ctx *cli.Context) {
	providers := make(map[string]*provider.Options)
	config := configPb.NewConfigService("go.micro.config", client.New(ctx))

	// todo: find a better way of accessing the config service
	req := &configPb.ReadRequest{Namespace: "global", Path: "micro.auth.providers"}
	if rsp, err := config.Read(context.TODO(), req); err == nil {
		var data map[string]string
		json.Unmarshal([]byte(rsp.Change.ChangeSet.Data), &data)

		for k, v := range data {
			var opts *provider.Options
			json.Unmarshal([]byte(v), &opts)
			providers[k] = opts
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	defer w.Flush()

	fmt.Fprintln(w, strings.Join([]string{"Name", "ClientID", "ClientSecret", "Endpoint", "Redirect", "Scope"}, "\t"))
	for name, opts := range providers {
		if !ctx.Bool("secrets") {
			opts.ClientID = "[redacted]"
			opts.ClientSecret = "[redacted]"
		}

		if opts.Type == "oauth" {
			opts.Redirect = "/auth/" + name + "/verify"
			opts.AuthURL = auth.OauthProviders[name].AuthURL
			name = opts.Type + "/" + name

			if len(opts.Scope) == 0 {
				opts.Scope = "default"
			}
		}

		if opts.Type == "basic" {
			name = "basic"
			opts.ClientID = "n/a"
			opts.ClientSecret = "n/a"
			opts.AuthURL = "n/a"
			opts.Redirect = "n/a"
			opts.Scope = "n/a"
		}

		fmt.Fprintln(w, strings.Join([]string{name, opts.ClientID, opts.ClientSecret, opts.AuthURL, opts.Redirect, opts.Scope}, "\t"))
	}
}

func createProvider(ctx *cli.Context) {
	if ctx.Args().Len() != 1 {
		fmt.Println("Invalid args, example use: micro auth create provider oauth/google")
		return
	}

	config := configPb.NewConfigService("go.micro.config", client.New(ctx))

	if ctx.Args().First() == "basic" {
		bytes, _ := json.Marshal(&provider.Options{Type: "basic"})

		_, err := config.Update(context.TODO(), &configPb.UpdateRequest{
			Change: &configPb.Change{
				Namespace: "global",
				Path:      "micro.auth.providers.basic",
				ChangeSet: &configPb.ChangeSet{
					Data:      string(bytes),
					Format:    "json",
					Source:    "cli",
					Timestamp: time.Now().Unix(),
				},
			},
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Auth provider basic created")
		return
	}

	var name string
	if comps := strings.Split(ctx.Args().First(), "/"); comps[0] != "oauth" || len(comps) != 2 {
		fmt.Println("Unknown provider type, supported types: basic or oauth")
	} else {
		name = comps[1]
	}

	// check to see if it's a supported oauth provider, e.g. google
	if _, ok := auth.OauthProviders[name]; !ok {
		fmt.Println("Unsupported oauth provider")
		return
	}

	if len(ctx.String("client_id")) == 0 {
		fmt.Println("Missing required oauth paramater: client_id")
		return
	}
	if len(ctx.String("client_secret")) == 0 {
		fmt.Println("Missing required oauth paramater: client_secret")
		return
	}

	bytes, _ := json.Marshal(&provider.Options{
		Type:         "oauth",
		ClientID:     ctx.String("client_id"),
		ClientSecret: ctx.String("client_secret"),
		Scope:        ctx.String("scope"),
	})

	_, err := config.Update(context.TODO(), &configPb.UpdateRequest{
		Change: &configPb.Change{
			Namespace: "global",
			Path:      "micro.auth.providers." + name,
			ChangeSet: &configPb.ChangeSet{
				Data:      string(bytes),
				Format:    "json",
				Source:    "cli",
				Timestamp: time.Now().Unix(),
			},
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Auth provider %v created\n", name)
}

func deleteProvider(ctx *cli.Context) {
	if ctx.Args().Len() == 0 {
		fmt.Println("Invalid args, example use: micro auth delete provider oauth/google")
	}

	// remove type from the name
	name := strings.TrimPrefix(ctx.Args().First(), "oauth/")

	config := configPb.NewConfigService("go.micro.config", client.New(ctx))
	_, err := config.Delete(context.TODO(), &configPb.DeleteRequest{
		Change: &configPb.Change{
			Namespace: "global",
			Path:      "micro.auth.providers." + name,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Auth provider %v deleted\n", ctx.Args().First())
}
