-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_kind AS ENUM ('coach', 'trainee');

CREATE TABLE users
(
    user_id       uuid        NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    email         VARCHAR     NOT NULL,
    password_hash VARCHAR     NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE authorizations
(
    identifier  VARCHAR(32) NOT NULL PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users,
    created_at  timestamptz NOT NULL DEFAULT now(),
    valid_until timestamptz NOT NULL,
    logout_at   timestamptz NULL     DEFAULT NULL
);

CREATE TABLE devices
(
    authorization_identifier VARCHAR(32) REFERENCES authorizations,
    ip_address               VARCHAR(15) NOT NULL DEFAULT '',
    browser                  VARCHAR     NOT NULL DEFAULT '',
    os                       VARCHAR     NOT NULL DEFAULT '',
    device_model             VARCHAR     NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE devices;
DROP TABLE authorizations;
DROP TABLE users;
DROP TYPE user_kind;