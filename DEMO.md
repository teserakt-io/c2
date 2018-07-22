# Demo setup


## 1. Have an MQTT broker running

For local broker, e.g. on macOS:

```
brew install mosquitto
mosquitto -c /usr/local/etc/mosquitto/mosquitto.conf
```

Broker will then run on localhost:1883.

Backend and demo app will by default use test.mosquitto.org:1883.

For a list of public brokers see <https://github.com/mqtt/mqtt.github.io/wiki/public_brokers>.

## 2. Start the backend

Run `c2backend`, it will list for gRPC connections on port 50051 by default. 
See [c2backend/parameters.go][c2backend/parameters.go] for parameters, where QoS is set to 2 (that is, commands will be sent to clients with QoS 2).


## 3. Start the demo client application

In another reminal, run `e4demoapp` in subscription mode, subscribing to the topic that the client will receive commands to.
For this demo the id and key are hardcoded (see constants in [e4demoapp/app.go](e4demoapp/app.go)).
QoS must be the same as the one used by C2 to send commands:

```
e4demoapp -action sub -qos 2 -topic "E4/2dd31f9cbe1ccf9f3f67520a8bc9594b7fe095ea69945408b83c861021372169" -num 10
```

## 4. Send commands to the client 

Using `c2cli`:

Add new client to C2, then add a new topic to C2, then send the topic key to the client (using the ID alias rather than raw value):

```
c2cli -c nc -id testit -pwd testpwd
c2cli -c nt -topic testtopic
c2cli -c ntc -id testid -topic testtopic
```


