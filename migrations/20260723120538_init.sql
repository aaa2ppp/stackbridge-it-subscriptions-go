-- +goose Up
CREATE TABLE subscription (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_name VARCHAR(255),
    price bigint NOT NULL,
    user_id uuid NOT NULL,
    start_date date NOT NULL
);

-- +goose Down
DROP TABLE subscription;
