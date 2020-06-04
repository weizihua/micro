package handler

import (
	"context"
	"time"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/errors"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/service"
	pb "github.com/micro/go-micro/v2/registry/service/proto"
	"github.com/micro/micro/v2/internal/namespace"
)

type Registry struct {
	// service id
	Id string
	// the publisher
	Publisher micro.Publisher
	// internal registry
	Registry registry.Registry
	// auth to verify clients
	Auth auth.Auth
}

func ActionToEventType(action string) registry.EventType {
	switch action {
	case "create":
		return registry.Create
	case "delete":
		return registry.Delete
	default:
		return registry.Update
	}
}

func (r *Registry) publishEvent(action string, service *pb.Service) error {
	// TODO: timestamp should be read from received event
	// Right now registry.Result does not contain timestamp
	event := &pb.Event{
		Id:        r.Id,
		Type:      pb.EventType(ActionToEventType(action)),
		Timestamp: time.Now().UnixNano(),
		Service:   service,
	}

	log.Debugf("publishing event %s for action %s", event.Id, action)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.Publisher.Publish(ctx, event)
}

// GetService from the registry with the name requested
func (r *Registry) GetService(ctx context.Context, req *pb.GetRequest, rsp *pb.GetResponse) error {
	// get the services in the default (shared) namespace
	services, err := r.Registry.GetService(req.Service, registry.GetDomain(namespace.DefaultNamespace))
	if err != nil && err != registry.ErrNotFound {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	// get the services in the current namespace, e.g. the "foo" namespace. name
	// includes the namespace as the prefix, e.g. 'foo/go.micro.service.bar'
	if ns := namespace.FromContext(ctx); ns != namespace.DefaultNamespace {
		srvs, err := r.Registry.GetService(req.Service, registry.GetDomain(ns))
		if err != nil && err != registry.ErrNotFound {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		services = append(services, srvs...)
	}

	// get the services in the requested domain. TODO: authenticate this so only the server can
	// request access to any namespace
	if len(req.Domain) > 0 && req.Domain != namespace.DefaultNamespace {
		srvs, err := r.Registry.GetService(req.Service, registry.GetDomain(req.Domain))
		if err != nil && err != registry.ErrNotFound {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		services = append(services, srvs...)
	}

	// return not found of no results are found
	if len(services) == 0 {
		return errors.NotFound("go.micro.registry", registry.ErrNotFound.Error())
	}

	// serialize the services
	for _, srv := range services {
		rsp.Services = append(rsp.Services, service.ToProto(srv))
	}
	return nil
}

// Register a service
func (r *Registry) Register(ctx context.Context, req *pb.Service, rsp *pb.EmptyResponse) error {
	opts := []registry.RegisterOption{
		registry.RegisterDomain(namespace.FromContext(ctx)),
	}

	if req.Options != nil {
		ttl := time.Duration(req.Options.Ttl) * time.Second
		opts = append(opts, registry.RegisterTTL(ttl))
	}

	service := service.ToService(req)
	if err := r.Registry.Register(service, opts...); err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	// publish the event
	go r.publishEvent("create", req)

	return nil
}

// Deregister a service
func (r *Registry) Deregister(ctx context.Context, req *pb.Service, rsp *pb.EmptyResponse) error {
	opts := []registry.DeregisterOption{
		registry.DeregisterDomain(namespace.FromContext(ctx)),
	}

	service := service.ToService(req)
	if err := r.Registry.Deregister(service, opts...); err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	// publish the event
	go r.publishEvent("delete", req)

	return nil
}

// ListServices returns all the services
func (r *Registry) ListServices(ctx context.Context, req *pb.ListRequest, rsp *pb.ListResponse) error {
	// get the services in the default domain
	services, err := r.Registry.ListServices(registry.ListDomain(namespace.DefaultNamespace))
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	// get the services in the requested domain if it isn't the default
	if ns := namespace.FromContext(ctx); ns != namespace.DefaultNamespace {
		srvs, err := r.Registry.ListServices(registry.ListDomain(ns))
		if err != nil {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		services = append(services, srvs...)
	}

	// list the services in the requested domain. TODO: authenticate this so only the server can
	// request access to any namespace
	if len(req.Domain) > 0 && req.Domain != namespace.DefaultNamespace {
		srvs, err := r.Registry.ListServices(registry.ListDomain(req.Domain))
		if err != nil && err != registry.ErrNotFound {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		services = append(services, srvs...)
	}

	// serialize the services
	for _, srv := range services {
		rsp.Services = append(rsp.Services, service.ToProto(srv))
	}

	return nil
}

// Watch a service for changes
func (r *Registry) Watch(ctx context.Context, req *pb.WatchRequest, rsp pb.Registry_WatchStream) error {
	// exit is closed when
	exit := make(chan bool)

	// master channel all events will flow through. since we could be combinining two channels below
	// this simplifies things.
	results := make(chan *registry.Result)

	// watch the default (shared) namespace
	opts := []registry.WatchOption{
		registry.WatchService(req.Service),
		registry.WatchDomain(namespace.DefaultNamespace),
	}

	watcher, err := r.Registry.Watch(opts...)
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}
	defer watcher.Stop()
	go func() {
		c := watcher.Chan()

		for {
			select {
			case <-exit:
				return
			case ev := <-c:
				results <- ev
			}
		}
	}()

	// watch the custom namespace if specified
	if ns := namespace.FromContext(ctx); ns != namespace.DefaultNamespace {
		opts = []registry.WatchOption{
			registry.WatchService(req.Service),
			registry.WatchDomain(ns),
		}
		watcher2, err := r.Registry.Watch(opts...)
		if err != nil {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		defer watcher2.Stop()
		go func() {
			c := watcher2.Chan()

			for {
				select {
				case <-exit:
					return
				case ev := <-c:
					results <- ev
				}
			}
		}()
	}

	for {
		select {
		case r, ok := <-results:
			// the results channel has closed
			if !ok {
				close(exit)
				return nil
			}

			// send the results from the channel to the stream
			err := rsp.Send(&pb.Result{Action: r.Action, Service: service.ToProto(r.Service)})
			if err != nil {
				close(exit)
				return errors.InternalServerError("go.micro.registry", err.Error())
			}
		case <-ctx.Done():
			// the context has finished
			close(exit)
			return nil
		}
	}
}
