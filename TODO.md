
#DONE

* test encode
* db status when starting
* storage abstraction
* tech for storage of topic->clients and client->topics
* mqtt integration
* cli with flags parsing
* support for all commands wrt db
* basic tester using cli 
* e4client
* document exported functions etc.
* demo app (add QoS to commands)
* go vet, go lint, goimports
* test persistent storage
* fix key length bug (size in db is 32?)
* C2 DB with topics instead of topichashes (revise specs)
* single db mode with data type signalling
* yaml config using https://github.com/spf13/viper, see flaki examples
* basic http server
* finish client persistence storage (write to disk every change)
* generate test vectors for encrypt() 
* moar tests of e4 ops
* refactor ops: make interface agnostic, with wrappers for gRPC and HTTP
* demo client command processing
* fix specs (key size)
* REST endpoints for frontend

# WIP

* GET endpoints
* set local GOPATH layout
* use dep

# TODO

* use same syntax as e4cli
* let C2/c2cli support sending of protect messages like another client
* test client encryption/decryption in demoapp

* make diagram using https://draw.io/
* interactive CLI with https://github.com/manifoldco/promptui
* fine-tune MQTT client options

# FUTURE

* 512b -> 256b key, cf mjos suggestion
* middleware for monitoring/tracing (Jaeger, Sentry, etc.), see go-kit
* monitoring of all topics' messages (c2monitor service)
* secure grpc + https
* binary packaging
    - binary in /opt/e4/, db in /var/lib/e4/db/
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
