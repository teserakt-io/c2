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

The CI automatically push docker images of C2 after each successfull builds and for each branches.
C2 from theses can be started like so:
```
# Replace <BRANCH_NAME> with the actual branch you want to pull the image from, like master, or devel, or tag...
docker run -it --rm -v $(pwd)/configs:/opt/e4/configs -p 5555:5555 -p 8888:8888 registry.gitlab.com/teserakt/c2:<BRANCH_NAME>
```

It just require a volume to the configs folder (Depending on your configuration, you may also need to get another volumes for the certificate and keys if they're not in the configs folder) and the ports for the GRPC and HTTP api (which can be independantly removed if not used)

List of available C2 images: https://gitlab.com/Teserakt/c2/container_registry

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


# Gitlab registry

CI will auto build docker images for devel branch. To be able to pull them, you must firstr login to the gitlab registry.
For this you first need to generate a personnal access token on gitlab, with the `api` scope:
- https://gitlab.com/profile/personal_access_tokens

Prior to use the `docker login` command, we need to configure the docker daemon to use a secret store. Otherwise tokens will get stored in clear in configuration file. (see https://docs.docker.com/engine/reference/commandline/login/#credentials-store for full reference)

First, install the docker-credential-helpers (the install instructions from their README are outdated, so you can follow those instead...):
```
go get github.com/docker/docker-credential-helpers...
cd $GOPATH/src/github.com/docker/docker-credential-helpers/
# for DBus
make secretservice
# for OSX keychain
make osxkeychain
cp bin/* $GOPATH/bin/
```

Create or append to `.docker/config.json`:

*DBus secret service:*
```
"credsStore": "secretservice",
```
*OSX keychain:*
```
"credsStore": "osxkeychain",
```

If you're already logged to a docker registry, remember to run `docker logout` first.
From here, run
```
docker login registry.gitlab.com
```
and enter your gitlab email and the personnal token as password.
It should display `Login Succeeded`. You can check it didn't stored clear password with:
```
cat .docker/config.json
# It should have:
# "auths": {
#		"registry.gitlab.com": {}
# }
# If wrong, it will show:
# "auths": {
#		"registry.gitlab.com": {"auth": "<b64 string with username/password>"}
# }
# Logout, check config & helpers installation, and retry login again
```
