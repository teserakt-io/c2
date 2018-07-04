# e4go

Suite of Go libraries and applications implementing E4.

Install dependencies:

```
go get github.com/eclipse/paho.mqtt.golang
go get github.com/dgraph-io/badger
go get github.com/miscreant/miscreant/go
go get golang.org/x/crypto/sha3
go get google.golang.org/grpc
```

Test by running `test.sh`.

# Components

* [c2backend](c2backend/): C2 back-end application, with gRPC server (for CLI) and HTTP server (for web UI). Key database uses [badger](https://github.com/dgraph-io/badger), but other storages can be plugged.

* [c2cli](c2cli/): CLI for C2, supporting various commands to implement the functionalities specified in the [E4 specification](https://gitlab.com/Teserakt/documentation/blob/master/E4.md).

* [c2proto](c2proto/): Protocol buffers format specification and Go package generated.

* [e4client](e4client/): Package to implement E4 client functionalities, with persistent storage of the client state.

* [e4common](e4common/): Package implementing functionalities common to back-end and client, such as encryption.

* [e4demoapp](e4demoapp/): Example client application using e4client to receive commands, encrypt messages published, and decrypt messages received.

## Go resources

https://peter.bourgon.org/go-best-practices-2016/

https://peter.bourgon.org/go-in-production/

https://peter.bourgon.org/go-for-industrial-programming/

https://12factor.net/

https://github.com/bahlo/go-styleguide/blob/master/README.md

http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/


Layout

* https://github.com/golang-standards/project-layout/blob/master/README.md

* https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1

* https://medium.com/golang-learn/go-project-layout-e5213cdcfaa2
