# E4 INTERNAL Documentation

[TOC]

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


# Go resources

https://peter.bourgon.org/go-best-practices-2016/
https://peter.bourgon.org/go-in-production/
https://peter.bourgon.org/go-for-industrial-programming/
https://12factor.net/
https://github.com/bahlo/go-styleguide/blob/master/README.md
http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/
http://www.ru-rocker.com/2017/02/17/micro-services-using-go-kit-http-endpoint/

dbs:

* https://github.com/jinzhu/gorm  / http://doc.gorm.io/
http:
* https://github.com/gin-gonic/gin
Layout
* https://github.com/golang-standards/project-layout/blob/master/README.md
* https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1
* https://medium.com/golang-learn/go-project-layout-e5213cdcfaa2


# E4: C2 database documentation and notes

TODO: intro, general restructuring

## E4 deployment

We use the most logical layout:

 * We create a database e4 and a role e4 to own it.
 * We create a schema and login-capable role named `e4_c2_$name`, where $name 
   is a desired instance name. The login role should have its search path 
   set to the relevant schema; the login role should also have its database 
   set to e4.

The logic behind this choice is as follows:

 * Since e4 owns the database, the company can back up all e4-specific 
   services at once.
 * Multiple instances of the C2 can be run easily - different C2s live in 
   different schemas.
 * We have two options for a multi-tenant "cloud" option: one database per 
   customer (probably best for backup/restore) or one whole database for all 
   using further schema restrictions.


## PostgreSQL considerations

### Schema vs Schema

PostgreSQL overloads the word "schema" - it means both "database layout" 
and "namespace" in PostgreSQL. We use the namespace sense. Basic schema 
management tasks are [explained here][pg-schema-expl]

A compatibility schema exists called "public"; this has the same name as the 
role "PUBLIC", which means "all roles". Database administrators should 
lock down their public schema with:

    REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC.

See [locking down public schema][pg-pubschema-lock].

Recommend doing this on development machines as we will use a schema `E4` and 
avoid any writes to other schemas (thus, this lockdown acts as a bug finding 
technique).

### Schema vs Database 

Databases are again a different concept and are collections of tables and can 
contain multiple schema. 

Cross database queries are NOT possible according to online documentation. Thus 
databases should be used only for datasets that are never going to be related. 

### Users, groups and roles

PostgreSQL does not have a concept of users or groups separately. Instead, it 
uses the concept of a role, which is both a user and a group. Individual roles 
may inherit permissions from other roles of which they are a part. 

A role may be assigned `LOGIN`. In this case, the role acts like a user in that 
it is valid for primary connection to the database. 

Roles are highly privileged if they have the `SUPERUSER` or `REPLICATION` 
privileges.

### Authentication

The 90s called and want their authentication mechanisms back.

Authentication is controlled by a file named `pg_hba.conf`, which contains 
user, group, host constraints for various authentication mechanisms. The 
fastest way to enable logins for users other than `postgres` is to modify this 
file to set network authentication to use `md5`, which is `md5` but twice for 
extra security (if you don't understand this snark, please talk to JP or Antony). 

My file looks like this:

```
local   all             all                                     peer
# IPv4 local connections:
host    all             all             127.0.0.1/32            md5
# IPv6 local connections:
host    all             all             ::1/128                 md5
# Allow replication connections from localhost, by a user with the
# replication privilege.
local   replication     all                                     peer
host    replication     all             127.0.0.1/32            ident
host    replication     all             ::1/128                 ident
```

This enables you to switch to the postgres user:

    sudo su - postgres
    psql 
    # done

while also enabling network logins, for example:

    sudo su - postgres    # not strictly necessary
    $ psql -U e4_c2_test -h localhost e4 -W
    Password for user e4_c2_test: ***
    # done

### psql help

 * `\d+ <tablename>` - describe a table.
 * `\dt+` - list tables.
 * `\connect <dbname>` - connect to db.
 * `\l` - list databases
 * `\z` - like `\dt+`, also describes sequences (indexes).
 * You can also do normal SQL of course. 

### PostgreSQL and SSL

Given `server.crt`, `server.key` and optionally `ca.crt`, 
which are: the certificate for the server, the private key for the server and 
an optional certificate authority bundle, as well as an optional 
`dhparams.pem` for finite field DH, we can configure a postgresql server 
to run its connections over SSL.

Place all of these in `/var/lib/pgsql/data`. Set their ownership to your 
postgres user and permissions to read only, for example

    chown postgres:postgres server.key
    chmod 0400 server.key

Now edit `postgresql.conf`, either located in this directory or possibly in 
`/etc` and configure these lines:

    ssl = on
    # ssl_ciphers = ... mozilla recommended cipher list ...
    ssl_ecdh_curve = 'prime256v1' #(P-256)
    ssl_dh_params_file = '/path/to/dhparams.pem'
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

### Multiplexing connections

TODO: idea from Alan.

### High Performance considerations

TODO: As we learn

## GORM KB

GORM is an ORM (Object-Relational-Mapping) library for Golang (hence, Go-ORM). 
The concept has existed for many years in object-oriented languages: a struct 
or class in the language is mapped (mostly through language reflection) to 
a table of data; manipulating data in the database generally corresponds to 
manipulating objects in the programming language.

GORM achieves this for golang using a DB struct to represent the connection and 
programmer-defined structs, which are annotated with gorm: annotations to 
describe sql constraints such as uniqueness.

### GORM model declaration

The GORM model structure looks like this:

```
type User struct {
  gorm.Model
  Name         string
  Age          sql.NullInt64
  Birthday     *time.Time
}
```

`gorm.Model` implicitly declares an `id` field as an `Int`, with the properties 
unique, primary key, not null. There are also three date fields, `created_at`,
`updated_at` and `deleted_at`, which will be updated whenever any of the 
verb actions are performed.

This enables **soft delete**. However, gorm works _just fine_ without this 
provided a primary key is supplied - in this case deletes are **hard** - the 
row is properly dropped.

### Slice queries

Query syntax looks something like this:

    var instance ModelType
    db.Where(...).Selector(&instance)

One selector option is to use strings:

    "E4ID=?", byteslice

This seems to fail; however the struct form works:

    &instance{FieldName: byteslice}

I have not explored why.

### Many to many considerations.

With backreferences (both structs reference each other, creating a true 
many-to-many relationship) actually querying and returning the relevant data 
across the relationship must be done with the `.Related()` construct. The 
`.Association()` only works for single values (adding or removing an 
association) and cannot return data or issue SQL queries with `OFFET` and 
`LIMIT` modifiers. An example of working code from `db.go` is:

    if err := s.db.Model(&topickey).Offset(offset).Limit(count).Related(&idkeys, "IDKeys").Error; err != nil {
		return nil, err
	}

   [pg-schema-expl]: https://severalnines.com/blog/postgresql-schema-management-basics
   [pg-pubschema-lock]: https://severalnines.com/blog/postgresql-privileges-and-security-locking-down-public-schema


# Demo setup

This assumes that the binaries have been successfully built using
script/build.sh and that the binaries are in bin/.

## 1. Run an MQTT broker

For example, a local broker on macOS can be installed and run as
follows:

```
brew install mosquitto
mosquitto -c /usr/local/etc/mosquitto/mosquitto.conf
```

The broker will then run on localhost:1883 by default.

Backend and demo client will by default use localhost:1883.

For a list of public brokers see <https://github.com/mqtt/mqtt.github.io/wiki/public_brokers>.


## 2. Run the C2 backend

Run bin/c2backend, it will list for gRPC connections on port 5555 by default. 

See configs/config.yaml for parameters.


## 3. Run an MQTT client

Run bin/mqe4client, which by default has the alias id `testid`, for
example with the following command:

```
mqe4client -action sub -broker tcp://localhost:1883 -num 10 -topic testtopic 
```

The client will then subscribe to topic `testtopic` in addition to `E4/<id>`.

By default the topic key will be derived from the password `testpwd`.


## 4. Send commands to the client 

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


