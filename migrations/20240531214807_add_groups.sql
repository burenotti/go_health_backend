-- +goose Up
-- +goose StatementBegin

CREATE TABLE groups
(
    group_id    uuid        NOT NULL PRIMARY KEY,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    coach_id    uuid        NOT NULL REFERENCES coaches_profiles ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL
);

CREATE TABLE invites
(
    invite_id   uuid        NOT NULL PRIMARY KEY,
    group_id    uuid        NOT NULL REFERENCES groups ON DELETE CASCADE,
    secret       varchar(20) NOT NULL,
    valid_until timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE invites_accept
(
    invite_id   uuid        NOT NULL REFERENCES invites ON DELETE CASCADE,
    trainee_id  uuid        NOT NULL REFERENCES trainees_profiles ON DELETE CASCADE,
    accepted_at timestamptz NOT NULL DEFAULT now()
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE invites_accept;
DROP TABLE invites;
DROP TABLE groups;

-- +goose StatementEnd
