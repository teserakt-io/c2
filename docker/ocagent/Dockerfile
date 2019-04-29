FROM golang:1.12 as builder

ARG OC_VERSION=0.1.6

RUN apt-get update && apt-get install -y bzr

RUN mkdir /app
RUN cd /app && git clone https://github.com/census-instrumentation/opencensus-service.git .
RUN cd /app && git checkout -b $OC_VERSION

RUN cd /app && make agent

FROM alpine AS ocagent

COPY --from=builder /app/bin/ocagent_linux /bin/ocagent_linux

VOLUME [ "/etc/ocagent/config.yaml" ]

# ocagent service default port
EXPOSE 55678
# zPages default port
EXPOSE 55679

ENTRYPOINT [ "ocagent_linux", "-c", "/etc/ocagent/config.yaml" ]
