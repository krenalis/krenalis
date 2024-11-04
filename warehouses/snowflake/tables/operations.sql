CREATE TABLE IF NOT EXISTS "_operations" (
    "id" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER,
    "operation" VARCHAR NOT NULL,
    "start_time" TIMESTAMP DEFAULT NULL,
    "end_time" TIMESTAMP DEFAULT NULL,
    "users_version" INT DEFAULT NULL,
    PRIMARY KEY ("id")
);