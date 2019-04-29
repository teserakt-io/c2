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


## Development

To set up a development environment for C2:

```bash
cp configs/config.yaml.example configs/config.yaml
openssl req -nodes -newkey rsa:2048 -keyout configs/c2-key.pem -x509 -sha256 -days 365 -out configs/c2-cert.pem -subj "/CN=localhost
```

The default configuration should work out of the box.

- Build with `scripts/build.sh`.
- Test with `scripts/test.sh`.
- Release with `scripts/release.sh` (in branch master only).
