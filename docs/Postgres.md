
# PostgreSQL considerations

## Database Security

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

## E4 deployment

### Databases and Schemas

We use the most logical layout:

 * We create a database e4 and a role e4 to own it.
 * We create a schema and login-capable role named e4_c2_$name, where $name 
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


   [pg-schema-expl]: https://severalnines.com/blog/postgresql-schema-management-basics
   [pg-pubschema-lock]: https://severalnines.com/blog/postgresql-privileges-and-security-locking-down-public-schema