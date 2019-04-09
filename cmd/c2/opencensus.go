package main

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func setupOpencensusInstrumentation(isProd bool) error {
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

	if isProd == false {
		// setting trace sample rate to 100%
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	return nil
}
