
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


# WIP

* FIX KEY LENGTH BUG (size in db is 32?)

* single db mode with data type signalling
* yaml config using https://github.com/spf13/viper, see flaki examples

# TODO

* generate test vectors for e4client
* REST endpoints for frontend, https://github.com/cloudtrust/flaki-service/blob/1.2/cmd/flakid.go#L535

* moar tests of dbops
* C2 DB with topics instead of topichashes (revise specs)
* apply https://github.com/golang/go/wiki/CodeReviewComments
* fine-tune MQTT client options

* interactive CLI with https://github.com/manifoldco/promptui

* monitoring of all topics' messages (c2monitor service)

* golint/go vet

# FUTURE

* secure grpc: encrypt + auth
* binary packaging
    - binary in /opt/e4/, db in /var/lib/e4/db/
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
