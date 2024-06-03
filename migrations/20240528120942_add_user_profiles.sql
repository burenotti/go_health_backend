-- +goose Up
CREATE TABLE trainees_profiles
(
    user_id    uuid REFERENCES users ON DELETE CASCADE PRIMARY KEY,
    first_name VARCHAR(20) NOT NULL DEFAULT '',
    last_name  VARCHAR(20) NOT NULL DEFAULT '',
    birth_date timestamptz NULL     DEFAULT NULL
);

CREATE TABLE coaches_profiles
(
    user_id          uuid REFERENCES users ON DELETE CASCADE PRIMARY KEY,
    first_name       VARCHAR(20) NOT NULL DEFAULT '',
    last_name        VARCHAR(20) NOT NULL DEFAULT '',
    birth_date       timestamptz NULL     DEFAULT NULL,
    years_experience INTEGER     NOT NULL DEFAULT 0,
    bio              TEXT        NOT NULL DEFAULT ''
);


-- +goose Down
DROP TABLE trainees_profiles;
DROP TABLE coaches_profiles;