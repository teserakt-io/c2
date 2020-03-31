![alt text](logo.png)

[![GoDoc][godoc-image]][godoc-url] ![Go](https://github.com/teserakt-io/c2/workflows/Go/badge.svg?branch=develop)


# c2

C2 back-end application, with gRPC server (for CLI) and HTTP server (for web UI).

A command line client is provided under `./bin/c2cli` to interact with the server.

The C2 server does provide endpoints to manage `clients` and `topics` keys, as well as `client-client` and `client-topic` relations.
Also, the C2 allows to publish E4 commands to the MQTT broker, allowing to control the managed clients' state, such as `NewClientKey`, or `ResetTopics` for example.
A complete list of available endpoints is available in the [api.proto](./api.proto) file.

For more details, you can check the [doc](./doc) folder.

## Running C2

### Start the services

```bash
./script/build.sh
docker-compose up -d
./bin/c2
```

This will start the MQTT broker (VerneMQ), Elasticsearch, Kibana, Prometheus, Jaeger, the OpenCensus Agent, and then start up C2.

### Services list

- [http://localhost:9200]: Elasticsearch endpoint
- [http://localhost:5601]: Kibana UI
- [http://localhost:16686]: Jaeger UI
- [http://localhost:9999]: zPages
- [http://localhost:9090]: Prometheus UI

### Run from Docker image

The C2 and C2 cli applications can be built in lightweight docker containers, with the requirement of having CGO disabled.

To build the docker images, just run:

```
# Build the c2 and c2cli binaries
CGO_ENABLED=0 ./scripts/build.sh
# Build docker images c2:devel and c2cli:devel
./scripts/docker-build.sh
```

Note that sqlite database isn't supported in docker as it requires CGO.

#### Start C2

```
# Replace <BRANCH_NAME> with the actual branch you want to pull the image from, like master, or devel, or tag...
docker run -it --rm  --name c2 -v $(pwd)/configs:/opt/e4/configs -p 5555:5555 -p 8888:8888 c2:<BRANCH_NAME>
```

It requires a volume to the configs folder (Depending on your configuration, you may also need to get another volumes for the certificate and keys if they're not in the configs folder) and the ports for the GRPC and HTTP api (which can be independently removed if not used)

#### Start C2Cli
```
# Replace <BRANCH_NAME> with the actual branch you want to pull the image from, like master, or devel, or tag...
docker run -it --rm \
    -v $(pwd)/configs/c2-cert.pem:/opt/c2/c2-cert.pem \
    -e C2_API_ENDPOINT=c2:5555 \
    -e C2_API_CERT=/opt/c2/c2-cert.pem \
    c2cli:<BRANCH_NAME> <command>
```

It requires a valid certificate C2 certificate. Both server endpoint and certificate path can be specified with the `-e` flag.

## Development

To set up a development environment for C2:

```bash
cp configs/config.yaml.example configs/config.yaml
# OpenSSL >= 1.1.1 only
# openssl req -nodes -newkey rsa:2048 -keyout configs/c2-key.pem -x509 -sha256 -days 365 -out configs/c2-cert.pem -subj "/CN=localhost" -addext "subjectAltName = 'IP:127.0.0.1'"
# Previous OpenSSL versions
openssl req  -nodes -newkey rsa:2048 -keyout configs/c2-key.pem -x509 -sha256 -days 365 -out configs/c2-cert.pem  -subj "/CN=localhost" -extensions san -config <(echo "[req]"; echo distinguished_name=req; echo "[san]"; echo subjectAltName=IP:127.0.0.1)
```

The default configuration should work out of the box.

- Build with `scripts/build.sh`.
- Test with `scripts/unittest.sh`.
- Run functional tests with `docker-compose up -d && ./scripts/test.sh`.
- Release with `scripts/release.sh` (in branch master only).

[godoc-image]: https://godoc.org/github.com/teserakt-io/c2?status.svg
[godoc-url]: https://godoc.org/github.com/teserakt-io/c2
