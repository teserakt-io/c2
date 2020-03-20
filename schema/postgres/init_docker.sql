-- Create a schema for a postgres running in docker (where db and user have already been setup)
CREATE SCHEMA IF NOT EXISTS e4_c2_test AUTHORIZATION e4_c2_test;
ALTER ROLE e4_c2_test SET search_path = e4_c2_test;
GRANT ALL ON SCHEMA e4_c2_test TO e4_c2_test;
