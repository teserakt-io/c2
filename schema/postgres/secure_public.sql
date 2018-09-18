
-- As documented in doc/, this removes access 
-- for all users to the public schema.
-- Each user will need a schema they can access configured 
-- as their default via search_path once you have done this.

REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC