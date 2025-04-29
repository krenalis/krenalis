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

INSERT INTO members (organization, name, avatar, email, password, created_at) VALUES (1, 'User', NULL, 'docker@open2b.com', '$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC', '2024-01-01 00:00:00.000000'); -- Password: foopass2