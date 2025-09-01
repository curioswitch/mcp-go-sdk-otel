package mcpotel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type config struct {
	propagator propagation.TextMapPropagator
}

func newConfig(opts []Option) *config {
	conf := &config{
		propagator: otel.GetTextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(conf)
	}
	return conf
}

// Option is a configuration option for middleware.
type Option interface {
	apply(conf *config)
}

// TextMapPropagator returns an Option indicating the propagator to use to
// extract and inject context in requests. If unset, the global TextMapPropagator
// will be used.
func TextMapPropagator(p propagation.TextMapPropagator) Option {
	return textMapPropagator{p: p}
}

type textMapPropagator struct {
	p propagation.TextMapPropagator
}

func (t textMapPropagator) apply(conf *config) {
	conf.propagator = t.p
}
