
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

# WIP

* refactor ops: make interface agnostic, with wrappers for gRPC and HTTP
* REST endpoints for frontend
    - GET all topics
    - GET all ids
* 512b -> 256b key, cf mjos suggestion

# TODO

* test client encryption/decryption in demoapp
* make diagram using https://draw.io/

* moar tests of dbops
* interactive CLI with https://github.com/manifoldco/promptui
* fine-tune MQTT client options

* QA: golint, go vet, https://github.com/golang/go/wiki/CodeReviewComments

# FUTURE

* monitoring/tracing (OpenCensus, Jaeger)
* go-kit? https://github.com/go-kit/kit
* monitoring of all topics' messages (c2monitor service)
* secure grpc: encrypt + auth
* https + auth
* binary packaging
    - binary in /opt/e4/, db in /var/lib/e4/db/
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
