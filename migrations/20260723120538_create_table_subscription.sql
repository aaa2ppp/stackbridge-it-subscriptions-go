-- +goose Up
CREATE TABLE subscription (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_name VARCHAR(255),
    price bigint NOT NULL,
    user_id uuid NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    created timestamptz NOT NULL DEFAULT NOW(),
    updated timestamptz NOT NULL DEFAULT NOW(),
    deleted timestamptz
);

-- +goose Down
DROP TABLE subscription;
