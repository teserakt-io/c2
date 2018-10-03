# E4: C2 backend server documentation

C2 back-end server, run from bin/c2backend.

Configuration file in configs/config.yaml.

TODO: restructure, add content, etc.

## Database setup

TODO: Postgres init, TLS

## Running and using C2

This assumes that the binaries have been successfully built using
script/build.sh and that the binaries are in bin/.

### 0. Run the database service

TODO

### 1. Run an MQTT broker

For example, a local broker on macOS can be installed and run as
follows:

```
brew install mosquitto
mosquitto -c /usr/local/etc/mosquitto/mosquitto.conf
```

The broker will then run on localhost:1883 by default.

Backend and demo client will by default use localhost:1883.

For a list of public brokers see <https://github.com/mqtt/mqtt.github.io/wiki/public_brokers>.


### 2. Run the C2 backend

Run bin/c2backend, it will list for gRPC connections on port 5555 by default. 

See configs/config.yaml for parameters.


### 3. Run an MQTT client

Run bin/mqe4client, which by default has the alias id `testid`, for
example with the following command:

```
mqe4client -action sub -broker tcp://localhost:1883 -num 10 -topic testtopic 
```

The client will then subscribe to topic `testtopic` in addition to `E4/<id>`.

By default the topic key will be derived from the password `testpwd`.


### 4. Send commands to the client 

Run bin/c2cli to send commands or messages to the client.
For example 
Add new client to C2, then add a new topic to C2, then send the topic key to the client (using the ID alias rather than raw value):

```bash
# register the client in the C2 db
c2cli -c nc -id testit -pwd testpwd

# register a new topic in the C2 db
c2cli -c nt -topic testtopic

# tell C2 to provision this topic's key to the client
c2cli -c ntc -id testid -topic testtopic

# send an encrypted message under the topic testtopic 
c2cli -c sm -topic testtopic -m "hello testtid!"
```



## APIs

Serves gRPC on TCP 5555, receiving `C2Request` objects as protobuf
messages as per api/c2.proto.

Serves HTTP on TCP 8888, receiving requests to the following endpoints:

E4 C2 API:

* POST /e4/client/{id}/key/{key}: `new_client(id, key)`

* DELETE /e4/client/{id}: `remove_client(id)`

* PUT /e4/client/{id}/topic/{topic}: `new_topic_client(id, topic)`

* DELETE /e4/client/{id}/topic/{topic}: `remove_topic_client(id, topic)`

* PUT /e4/client/{id}: `reset_client(id)` 

* POST /e4/topic/{topic}: `new_topic(topic)`

* DELETE /e4/topic/{topic}: `remove_topic(topic)` 

* PATCH /e4/client/{id}/: `new_client_key(id)` 

Other endpoints:

* GET /e4/topic/: lists of all topics

* GET /e4/client/: lists all client ids

* GET /e4/client/{id}: lists the topics support by id

* GET /e4/topic/{topic}: lists the ids supporting topic

## Security

TODO: TLS/CA etc.
