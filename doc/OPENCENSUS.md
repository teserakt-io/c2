# OpenCensus setup

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
UI then accessible on localhost:16686.

## Monitoring: Prometheus

As monitoring system, currently using Prometheus. Install and run
following the instructions on
<https://opencensus.io/codelabs/prometheus/> and
<https://prometheus.io/docs/introduction/first_steps/>.

Run e.g. as follows from ocagent directory:

``` 
$ prometheus --config.file=prom.yaml
``` 

## Collector: OpenCensus agent (ocagent)

OpenCensus agent used as a proxy to exporters/receivers:
<https://github.com/census-instrumentation/opencensus-service>.
For testing, run locally from
$GOPATH/src/github.com/census-instrumentation/opencensus-service.

Config files used:

config.yaml
```
receivers:
  opencensus:
    address: "127.0.0.1:55678"

exporters:
  jaeger:
    collector_endpoint: "localhost:14267"

  prometheus:
    namespace: "tesserakt"
    address: "localhost:9998"

zpages:
    port: 9999

``` 

prom.yaml:
``` 
global:
  scrape_interval: 10s

  external_labels:
    monitor: 'teseraktagent'

scrape_configs:
  - job_name: 'teseraktagent'

    scrape_interval: 10s

    static_configs:
      - targets: ['localhost:9998']
``` 
