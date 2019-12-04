package analytics

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// SetupObservability will configure the various observers for C2.
// currently register an opencensus exporter
func SetupObservability(ocAgentAddr string, ocAgentSampleAll bool) error {
	oce, err := ocagent.NewExporter(
		// TODO: (@odeke-em), enable ocagent-exporter.WithCredentials option.
		ocagent.WithInsecure(),
		ocagent.WithServiceName("c2"),
		ocagent.WithAddress(ocAgentAddr),
	)

	if err != nil {
		return fmt.Errorf("failed to create the OpenCensus Agent exporter: %v", err)
	}

	// and now finally register it as a Trace Exporter
	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)

	// setting trace sample rate to 100%
	if ocAgentSampleAll {
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	return nil
}
