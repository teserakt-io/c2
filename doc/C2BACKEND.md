# E4: C2 backend server documentation

C2 back-end server, run from bin/c2backend.

Configuration file in configs/config.yaml.

TODO: restructure, add content, etc.

## Database setup

The C2 backend uses relational databases as its datastore. You 
can use sqlite3 (which is supported for testing/demos) or you 
can deploy fully using postgresql (supported for production).

In all cases you must set

    db-encryption-passphrase: somevalue

This value cannot be empty.

### SQLite3

It is sufficient to set only two configuration values:

    db-type: sqlite3
    db-file: /path/to/e4c2.sqlite

to specify the SQLite3 file.

### PostgreSQL

PostgreSQL setup is a little trickier. You must install 
postgresql for your platform. On RedHat systems this might look 
like

    yum install postgresql-server
    postgresql-setup --initdb

The rationale and details for the postgresql database design are explained in DB.md and will not be repeated here. A working 
setup schema is provided in `schemas/postgres/init.sql`. On 
linux, run:

    sudo su - postgres

to switch to the postgresql user and then

    psql < /path/to/e4go/schemas/postgres/init.sql

to execute the script (you can optionally run the secure_public.sql too). This will create a database `e4`, a 
user `e4_c2_test` and a matching schema, set everything 
up correctly etc.

These settings can then appear in your `config.yaml`:

    db-type: postgres
    db-username: e4_c2_test
    db-password: teserakte4
    db-logging: 0

The last line toggles database logging. Set it to 1 for 
development diagnostics.

You also need to configure your database server correctly 
for access. First, go to `/var/lib/pgsql/data` (this path may 
different for your distribution).

**Note on the homebrew install** The homebrew install uses `trust` 
authentication in `pg_hba.conf`. You do not need to make any changes to 
`pg_hba.conf` as a result - postgresql will accept any username provided it 
exists and any password without checking it. This is obviously _not suitable 
for production_ but is fine for development environments.

**If you do not want to configure SSL** then things are relatively straightforward. You need to edit `pg_hba.conf` to 
configure user authentication:

    # IPv4 local connections:
    host    all             all             127.0.0.1/32            md5
    # IPv6 local connections:
    host    all             all             ::1/128                 md5

This allows network logins using passwords over the network. You can then add

    db-secure-connection: insecure

to your configuration file. Connections will be plaintext over 
port 5432.

**If you wish to deploy SSL** things are only slightly more involved. You 
need to obtain three files, `server.crt`, `server.key` and optionally `ca.crt`, 
which are: the certificate for the server, the private key for the server and 
an optional certificate authority bundle.

Place all of these in `/var/lib/pgsql/data`. Set their ownership to your 
postgres user and permissions to read only, for example

    chown postgres:postgres server.key
    chmod 0400 server.key

Now edit `postgresql.conf`, either located in this directory or possibly in 
`/etc` and configure these lines:

    ssl = on
    # ssl_ciphers = ... mozilla recommended cipher list ...
    ssl_prefer_server_ciphers = on
    ssl_cert_file = 'server.crt'
    ssl_key_file = 'server.key'
    ssl_ca_file = 'ca.crt'

You can add additional ssl configuration as required (curve choices, dh 
parameters etc) for production environments. In your `pg_hba.conf` file you 
must now set:

    # IPv4 local connections:
    hostssl    all             all             127.0.0.1/32            md5
    # IPv6 local connections:
    hostssl    all             all             ::1/128                 md5

You will also be required to set `hostssl` for replication entries if deploying 
a server with replication (not required for development).

You can restart the server (`systemctl restart postgresql`) and then try

    sudo su - postgres
    psql -U e4_c2_test -h 127.0.0.1 -W e4
    Password: (type it)

and connect to the database.

If your certificate is self signed, you can set

    db-secure-connection: selfsigned

If your certificate is signed by a known certificate authority from the system 
store, you can instead use:

    db-secure-connection: yes

You will also need to set

    db-encryption-passphrase: somevalue

If you change, forget or lose this value you will lose access to any key 
material created in the database (all client keys and topic keys are encrypted).

## Running and using C2

This assumes that the binaries have been successfully built using
script/build.sh and that the binaries are in bin/.

### 0. Run the database service

See setting up the database above. You should verify it is running with the 
equivalent of `systemctl status postgresql` and you should attempt to connect 
with 

    psql -U e4_c2_test -h 127.0.0.1 -W e4

If you can connect and the postgresql prompt is

    e4=>

You are connected through the postgres-supplied client.

If you wish to run the database on boot, on Linux run

    systemctl enable postgresql

### 1. Run an MQTT broker

For example, a local broker on macOS can be installed and run as
follows:

```
brew install mosquitto
mosquitto -c /usr/local/etc/mosquitto/mosquitto.conf
```

The broker will then run on localhost:1883 by default.

On linux (redhat systems)

    yum install mosquitto
    systemctl start mosquitto

You can make this run on boot with:

    systemctl enable mosquitto

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

## Monitoring

C2 will subscribe to all the MQTT topics for which it generated a key, and will generate logs for each messages received, where a log will include payload, metadata, and enriched data.

We use Docker containers for E, L, and K applications, with startup scripts in scripts/, and configuration files in configs/.

These logs are to be processed by an ELK, configured as follows:


### Logstash

See https://www.elastic.co/guide/en/logstash/current/docker.html

### Elasticsearch

See https://www.elastic.co/guide/en/elasticsearch/reference/current/docker.html

Config: https://www.elastic.co/guide/en/kibana/6.6/settings.html

TODO: config, authentication


### Kibana

See https://www.elastic.co/guide/en/kibana/current/docker.html


