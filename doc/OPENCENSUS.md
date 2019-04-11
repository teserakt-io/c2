# Observability setup using OpenCensus agent

General basic instructions for metrics and tracing:

- <https://opencensus.io/quickstart/go/metrics/>
- <https://opencensus.io/quickstart/go/tracing/>

## Transports instrumentation

gRPC instrumentation as documented in:

- <https://opencensus.io/guides/grpc/go/>. 
- <https://opencensus.io/exporters/supported-exporters/go/jaeger/>

Can be added to both server and client, sending data to the same
exporter.

HTTP instrumentation as in:

- <https://opencensus.io/guides/http/go/net_http/server/>
- <https://opencensus.io/guides/http/go/net_http/client/>

## Tracing: Jaeger

As an exporter, currently using Jaeger in a Docker container. Install
and run following the instructions on
<https://opencensus.io/codelabs/jaeger/>.

UI accessible at <http://localhost:16686>.

## Monitoring: Prometheus

As monitoring system, currently using Prometheus. Install and run
following the instructions on
<https://opencensus.io/codelabs/prometheus/> and
<https://prometheus.io/docs/introduction/first_steps/>.

Run e.g. as follows (using the config file in c2/configs)

``` 
$ prometheus --config.file=configs/prometheus.yaml
``` 

UI accessible at <http://localhost:9090>.

## Collector: OpenCensus agent (ocagent)

OpenCensus agent used as a proxy to exporters/receivers:
<https://github.com/census-instrumentation/opencensus-service>.
For testing, run locally from e.g.:

$GOPATH/src/github.com/census-instrumentation/opencensus-service.

See configs/ocagent.yaml for the configuration file.

