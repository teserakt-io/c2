
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

# WIP

* yaml config using https://github.com/spf13/viper, see flaki examples
* http server
* REST endpoints for frontend, https://github.com/cloudtrust/flaki-service/blob/1.2/cmd/flakid.go#L535
* create interface Storage, instantiate with badger.go, element of C2
* finish client persistence storage (write to disk every change)

# TODO

* generate test vectors to encrypt() and protect()
* test client encryption/decryption
* structure as https://github.com/prometheus/prometheus?
* make diagram using https://draw.io/

* make config common to all apps
* moar tests of dbops
* interactive CLI with https://github.com/manifoldco/promptui
* fine-tune MQTT client options

* QA: golint, go vet, https://github.com/golang/go/wiki/CodeReviewComments

# FUTURE

* monitoring/tracing (OpenCensus, Jaeger)

* monitoring of all topics' messages (c2monitor service)
* secure grpc: encrypt + auth
* binary packaging
    - binary in /opt/e4/, db in /var/lib/e4/db/
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
