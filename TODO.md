
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


# WIP

* test persistent storage, integrate in demo
* fine-tune MQTT client options

# TODO

* REST endpoints for frontend, https://github.com/cloudtrust/flaki-service/blob/1.2/cmd/flakid.go#L535
* moar tests of dbops
* C2 DB with topics instead of topichashes (revise specs)
* <https://github.com/golang/go/wiki/CodeReviewComments#error-strings>

* monitoring of all topics' messages (c2monitor service)

# FUTURE

* secure grpc: encrypt + auth
* binary packaging
    - https://stackoverflow.com/a/29600086
    - https://stackoverflow.com/a/45003378
    - https://github.com/goreleaser/nfpm ?
