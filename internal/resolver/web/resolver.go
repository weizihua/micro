package web

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	res "github.com/micro/go-micro/v2/api/resolver"
	"github.com/micro/go-micro/v2/router"
	"github.com/micro/go-micro/v2/selector"
)

var re = regexp.MustCompile("^[a-zA-Z0-9]+([a-zA-Z0-9-]*[a-zA-Z0-9]*)?$")

type Resolver struct {
	// Namespace of the request, e.g. go.micro.web
	Namespace string
	// selector to choose from a pool of nodes
	Selector selector.Selector
	// router to lookup routes
	Router router.Router
}

func (r *Resolver) String() string {
	return "web/resolver"
}

// Resolve replaces the values of Host, Path, Scheme to calla backend service
// It accounts for subdomains for service names based on namespace
func (r *Resolver) Resolve(req *http.Request) (*res.Endpoint, error) {
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) < 2 {
		return nil, errors.New("unknown service")
	}

	if !re.MatchString(parts[1]) {
		return nil, res.ErrInvalidPath
	}

	// lookup the routes for the service
	routes, err := r.Router.Lookup(router.QueryService(r.Namespace + "." + parts[1]))
	if err != nil {
		return nil, err
	}

	// select the route to use
	route, err := r.Selector.Select(routes...)
	if err != nil {
		return nil, err
	}

	// we're done
	return &res.Endpoint{
		Name:   parts[1],
		Method: req.Method,
		Host:   route.Address,
		Path:   "/" + strings.Join(parts[2:], "/"),
	}, nil
}
