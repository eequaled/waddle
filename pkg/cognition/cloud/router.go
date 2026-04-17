package cloud

import (
	"context"
	"errors"
)

// Router selects between local and cloud providers based on configuration
type Router struct {
	local Provider
	cloud Provider
	mode  string // "local", "cloud", "hybrid"
}

func NewRouter(local, cloud Provider, mode string) *Router {
	if mode == "" {
		mode = "hybrid"
	}
	return &Router{
		local: local,
		cloud: cloud,
		mode:  mode,
	}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	provider, err := r.selectProvider()
	if err != nil {
		return nil, err
	}
	return provider.Complete(ctx, req)
}

func (r *Router) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	provider, err := r.selectProvider()
	if err != nil {
		return nil, err
	}
	return provider.Stream(ctx, req)
}

func (r *Router) Name() string {
	return "router"
}

func (r *Router) HealthCheck(ctx context.Context) error {
	var errs []error
	
	if r.local != nil {
		if err := r.local.HealthCheck(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if r.cloud != nil {
		if err := r.cloud.HealthCheck(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (r *Router) selectProvider() (Provider, error) {
	switch r.mode {
	case "local":
		if r.local == nil {
			return nil, errors.New("local mode requested but no local provider configured")
		}
		return r.local, nil
	case "cloud":
		if r.cloud == nil {
			return nil, errors.New("cloud mode requested but no cloud provider configured")
		}
		return r.cloud, nil
	case "hybrid":
		// In a real implementation, this might dynamically check hardware or request complexity.
		// For now, if cloud is available we prefer it for better reasoning, fallback to local.
		if r.cloud != nil {
			return r.cloud, nil
		}
		if r.local != nil {
			return r.local, nil
		}
		return nil, errors.New("no providers configured for hybrid mode")
	default:
		return nil, errors.New("invalid routing mode configured")
	}
}
