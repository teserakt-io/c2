# c2backend

C2 back-end server, see [configs/config.yaml](configs/config.yaml).

Serves gRPC on TCP 5555, receiving `C2Request` objects as protobuf
messages as per [c2.proto](https://gitlab.com/Teserakt/e4go/tree/master/c2proto).

Serves HTTP on TCP 8888, receiving requests to the following endpoints:

* POST /e4/client/{id}/key/{key}: `new_client(id, key)`

* DELETE /e4/client/{id}: `remove_client(id)`

* PUT /e4/client/{id}/topic/{topic}: `new_topic_client(id, topic)`

* DELETE /e4/client/{id}/topic/{topic}: `remove_topic_client(id, topic)`

* PUT /e4/client/{id}: `reset_client(id)` 

* POST /e4/topic/{topic}: `new_topic(topic)`

* DELETE /e4/topic/{topic}: `remove_topic(topic)` 

* PUT /e4/client/{id}/: `new_client_key(id)` 
