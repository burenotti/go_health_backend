-- +goose Up
-- +goose StatementBegin
ALTER TABLE devices
    ADD COLUMN authorization_id uuid NULL DEFAULT NULL;


ALTER TABLE devices
    DROP CONSTRAINT devices_authorization_identifier_fkey;


ALTER TABLE authorizations
    ADD COLUMN secret VARCHAR(32);


UPDATE authorizations
SET secret = identifier
WHERE TRUE;


ALTER TABLE authorizations
    ADD COLUMN authorization_id uuid NOT NULL DEFAULT uuid_generate_v4();


UPDATE devices d
SET authorization_id = (SELECT a.authorization_id
                        FROM authorizations a
                        WHERE a.identifier = d.authorization_identifier);

ALTER TABLE devices DROP COLUMN authorization_identifier;

ALTER TABLE authorizations
    DROP CONSTRAINT authorizations_pkey;


ALTER TABLE authorizations
    ADD CONSTRAINT authorizations_pk PRIMARY KEY (authorization_id);

ALTER TABLE devices
    ADD CONSTRAINT devices_authorization_id_fkey
        FOREIGN KEY (authorization_id) REFERENCES authorizations;

ALTER TABLE authorizations DROP COLUMN identifier;

-- +goose StatementEnd
