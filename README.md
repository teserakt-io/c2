# e4go

Go libraries and commands implementing E4 client and C2 functionalities.

Master Branch Status: [![master pipeline status](https://gitlab.com/Teserakt/e4go/badges/master/pipeline.svg)](https://gitlab.com/Teserakt/e4go/commits/master)

Develop Branch Status: [![develop pipeline status](https://gitlab.com/Teserakt/e4go/badges/develop/pipeline.svg)]
(https://gitlab.com/Teserakt/e4go/commits/develop)

* [c2backend](cmd/c2backend/): C2 back-end application, with gRPC server (for CLI) and HTTP server (for web UI). Key database uses [badger](https://github.com/dgraph-io/badger), but other storages can be plugged.

* [c2cli](cmd/c2cli/): CLI for C2, supporting various commands to implement the functionalities specified in the [E4 specification](https://gitlab.com/Teserakt/documentation/blob/master/E4.md).

* [c2proto](pkg/c2proto/): Protocol buffers format specification and Go package generated.

* [e4client](pkg/e4client/): Package to implement E4 client functionalities, with persistent storage of the client state.

* [e4common](pkg/e4common/): Package implementing functionalities common to back-end and client, such as encryption.

* [mqe4client](mqe4client/): Example client application using e4client to receive commands, encrypt messages published, and decrypt messages received.


Build with `scripts/build.sh`.

Test with `scripts/test.sh`.

Release with `scripts/release.sh` (in branch master only).

Demo following instructions in [DEMO.md](docs/DEMO.md).

