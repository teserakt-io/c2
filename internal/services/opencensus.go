package services

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// OpenCensus defines an OpenCensus service
type OpenCensus interface {
	Setup() error
}

type openCensus struct {
	isProd bool
}

var _ OpenCensus = &openCensus{}

// NewOpenSensus creates a new OpenCensus service
func NewOpenSensus(isProd bool) OpenCensus {
	return &openCensus{
		isProd: isProd,
	}
}

func (oc *openCensus) Setup() error {
	oce, err := ocagent.NewExporter(
		// TODO: (@odeke-em), enable ocagent-exporter.WithCredentials option.
		ocagent.WithInsecure(),
		ocagent.WithServiceName("c2"))

	if err != nil {
		return fmt.Errorf("failed to create the OpenCensus Agent exporter: %v", err)
	}

	// and now finally register it as a Trace Exporter
	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)

	if oc.isProd == false {
		// setting trace sample rate to 100%
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	return nil
}
