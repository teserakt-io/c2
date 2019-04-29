# syntax = docker/dockerfile:1.0-experimental

FROM golang:1.12 as buildenv

RUN mkdir -p ~/.ssh && echo "gitlab.com,35.231.145.151 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=" >> ~/.ssh/known_hosts

# Fix gitlab protocal to use ssh instead of https
RUN git config --global url."git@gitlab.com:".insteadOf "https://gitlab.com/"

RUN mkdir -p /build/
COPY go.mod /build/
COPY go.sum /build/

# put go mod downlaod in its own layer, this speed up the build by A LOT :)
RUN --mount=type=ssh cd /build/ && go mod download

COPY . /build/

RUN cd /build/ &&  ls -lah scripts/ && ./scripts/build.sh

FROM alpine AS c2

COPY --from=buildenv /build/bin/c2 /opt/e4/bin/c2
COPY --from=buildenv /build/configs/ /opt/e4/configs

# Fix glibc dependancy
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

# Redirect log file to stdout
RUN ln -s /dev/stdout /var/log/e4_c2.log

WORKDIR /opt/e4

CMD ["/opt/e4/bin/c2"]
