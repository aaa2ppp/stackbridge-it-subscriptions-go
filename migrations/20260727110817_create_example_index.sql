-- +goose Up
CREATE INDEX idx_subscription_user_id_end_date_active ON subscription (user_id, end_date) 
    WHERE deleted IS NULL;

-- +goose Down
DROP INDEX idx_subscription_user_id_end_date_active;
