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


## GORM KB

TODO: Gorm is...

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



   [pg-schema-expl]: https://severalnines.com/blog/postgresql-schema-management-basics
   [pg-pubschema-lock]: https://severalnines.com/blog/postgresql-privileges-and-security-locking-down-public-schema
