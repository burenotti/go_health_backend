-- +goose Up
CREATE TABLE metrics
(
    metric_id  uuid        NOT NULL PRIMARY KEY,
    trainee_id uuid        NOT NULL REFERENCES trainees_profiles,
    heart_rate int         NOT NULL,
    weight     int         NOT NULL,
    height     int         NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);


-- +goose Down
DROP TABLE metrics;
