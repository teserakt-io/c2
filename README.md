# c2

C2 back-end application, with gRPC server (for CLI) and HTTP server (for web UI).

## Running C2

### Start the services

```bash
./script/build.sh
docker-compose up -d
./bin/c2
```

This will boot up MQTT broker, ELK, prometheus, jaeger and oc-agent and then start up C2.

### Services list

- [http://localhost:9200]: elasticsearch endpoint
- [http://localhost:5601]: kibana UI
- [http://localhost:16686]: jaeger UI
- [http://localhost:9999]: zPages
- [http://localhost:9090]: prometheus UI

### Run from docker image

The CI automatically push docker images of C2 and C2Cli after each successful builds and for each branches.

List of available C2 and C2Cli images: https://console.cloud.google.com/gcr/images/teserakt-dev/EU/c2?project=teserakt-dev&authuser=1&gcrImageListsize=30

#### Start C2
```
# Replace <BRANCH_NAME> with the actual branch you want to pull the image from, like master, or devel, or tag...
docker run -it --rm  --name c2 -v $(pwd)/configs:/opt/e4/configs -p 5555:5555 -p 8888:8888 eu.gcr.io/teserakt-dev/c2:<BRANCH_NAME>
```

It just require a volume to the configs folder (Depending on your configuration, you may also need to get another volumes for the certificate and keys if they're not in the configs folder) and the ports for the GRPC and HTTP api (which can be independently removed if not used)

#### Start C2Cli
```
# Replace <BRANCH_NAME> with the actual branch you want to pull the image from, like master, or devel, or tag...
docker run -it --rm \
    -v $(pwd)/configs/c2-cert.pem:/opt/c2/c2-cert.pem \
    -e C2_API_ENDPOINT=c2:5555 \
    -e C2_API_CERT=/opt/c2/c2-cert.pem \
    eu.gcr.io/teserakt-dev/c2cli:<BRANCH_NAME> <command>
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
- Test with `scripts/test.sh`.
- Release with `scripts/release.sh` (in branch master only).


# GCP registry

CI will auto build docker images for all branch. To be able to pull them, you must first login to the GCP registry.
For this you first need to configure docker to be able to authenticate on GCP:
```
# Make sure your current active config points to teserakt-dev project
gcloud auth configure-docker
```

From here, you are able to `docker pull eu.gcr.io/teserakt-dev/<image>:<version>`
