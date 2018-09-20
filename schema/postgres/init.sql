
-- TODO: turn these scripts into something better.

-- Create an overall role "E4" to own the E4 database.
CREATE ROLE e4 WITH NOSUPERUSER NOCREATEROLE NOINHERIT NOLOGIN NOREPLICATION NOBYPASSRLS;

-- Create a database with en_US.UTF-8 locale; set owner to e4.
CREATE DATABASE e4 WITH OWNER=e4 LC_COLLATE="en_US.UTF-8" LC_CTYPE="en_US.UTF-8" ENCODING=UTF8 TEMPLATE=template0;
\connect e4;

-- Create a specific login role:
CREATE ROLE e4_c2_test WITH NOSUPERUSER NOCREATEROLE NOINHERIT LOGIN NOREPLICATION NOBYPASSRLS;
ALTER ROLE e4_c2_test WITH ENCRYPTED PASSWORD 'teserakte4';

-- Create a specific schema for that role to operate in.
CREATE SCHEMA IF NOT EXISTS e4_c2_test AUTHORIZATION e4_c2_test;
-- Configure specific role to login using specified schema by default:
ALTER ROLE e4_c2_test SET search_path = e4_c2_test;

-- Give overall role E4 access to the schema.
GRANT ALL ON SCHEMA e4_c2_test TO e4;
