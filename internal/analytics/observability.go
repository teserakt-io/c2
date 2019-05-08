package analytics

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// DeploymentMode describe the per environment analytics configuration modes
type DeploymentMode int

// List of available DeploymentMode
const (
	Development DeploymentMode = iota
	Production
)

// SetupObservability will configure the various observers for C2.
// currently register an opencensus exporter
func (m DeploymentMode) SetupObservability() error {
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

	switch m {
	case Production:
	default:
		// setting trace sample rate to 100%
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	return nil
}
