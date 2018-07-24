#DONE

* move config to ./config/
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
* GET endpoints
* let C2/c2cli support sending of protect messages like another client
    - tweak c2cli to send encrypted messages to clients
    - create new pbRequest type
    - c2cli command "send client topic message"
    - c2backend: handler to send message, needs e4common
* interactive CLI with ishell
* shorten ids à la git "8hexchars.."
* c2cli: show help if command missing
* add banner including version and date to backend, cli, client
* fix lints
* better logs (go-kit logs); only log in main, add handler to services
* set local GOPATH layout
* debug current setting
* use dep https://golang.github.io/dep/docs/daily-dep.html
* use goimports

# WIP

* c2cli: lsc and lst commands 
* make arch diagram using https://draw.io/

* slides with features (update master deck)
    - c2: structured logs for siem/splunk
    - c2: designed with people who done major e-commerce platform, HA
    - go, c/avr, java/bee clients
* demo:
    - mqtt.fail broker
    - screencast
    - bee client, go client

# TODO

* s/badger/cockroach
* more complete build.sh (arch, etc.)
* getIDs(topic), getTopics(id)
* fine-tune MQTT client options

# FUTURE

* 512b -> 256b key, cf mjos suggestion
* middleware for monitoring/tracing (Jaeger, Sentry, etc.), see go-kit
* monitoring of all topics' messages (c2monitor service)
* secure grpc + https
    -https://github.com/olivere/grpc-example/blob/master/tasks/server.go#L68
* binary packaging
    - binary in /opt/e4/, db in /var/lib/e4/db/
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
