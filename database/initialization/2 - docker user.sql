-- This SQL file is executed by Docker Compose during the initialization of the
-- PostgreSQL container.
-- 
-- It is essential that the name of this file follows alphabetically the name of
-- the file that creates the database, since this file must necessarily be
-- executed after the other. In this regard, see the documentation of the
-- PostgreSQL image for Docker:
-- 
-- https://hub.docker.com/_/postgres

TRUNCATE members;

INSERT INTO members (organization, name, avatar, email, password, created_at)
    SELECT id, 'User', NULL, 'docker@meergo.com', '$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC', now() at time zone 'utc' -- Password: foopass2
    FROM organizations;
