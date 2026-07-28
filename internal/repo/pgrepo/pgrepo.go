package pgrepo

import (
	"context"
	"fmt"
	"net"
	"strings"
	"subscriptions/internal/model"
	"subscriptions/internal/repo"
	"subscriptions/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepo pgxpool.Pool

func (repo *PGRepo) db() *pgxpool.Pool { return (*pgxpool.Pool)(repo) }

func Open(ctx context.Context, cfg repo.Config) (*PGRepo, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, cfg.DBName, cfg.User, cfg.Password, cfg.SSLMode)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return (*PGRepo)(pool), nil
}

func (repo *PGRepo) Close() {
	repo.db().Close()
}

func (r *PGRepo) Create(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	const sql = `INSERT INTO subscription (service_name, price, user_id, start_date, end_date) 
	VALUES ($1, $2, $3, $4, $5) RETURNING id, created, updated;`

	row := r.db().QueryRow(ctx, sql, req.ServiceName, req.Price, req.UserID, req.StartDate, req.EndDate)
	if err := row.Scan(
		&req.ID,
		&req.Created,
		&req.Updated,
	); err != nil {
		return model.Subscription{}, err
	}

	return req, nil
}

func (r *PGRepo) Get(ctx context.Context, id int64) (model.Subscription, error) {
	const sql = `SELECT id, service_name, price, user_id, start_date, end_date, created, updated
	FROM subscription WHERE id=$1 AND deleted IS NULL;`

	var resp model.Subscription

	row := r.db().QueryRow(ctx, sql, id)
	if err := row.Scan(
		&resp.ID,
		&resp.ServiceName,
		&resp.Price,
		&resp.UserID,
		&resp.StartDate,
		&resp.EndDate,
		&resp.Created,
		&resp.Updated,
	); err != nil {
		if err == pgx.ErrNoRows {
			err = model.ErrNotFound
		}
		return model.Subscription{}, err
	}

	return resp, nil
}

func (r *PGRepo) List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
	const sql = `SELECT id, service_name, price, user_id, start_date, end_date, created, updated
	FROM subscription 
	WHERE id > $1 AND start_date <= $3 AND $2 <= end_date AND deleted IS NULL %s
	ORDER BY id %s;`

	values := []any{req.AfterID, req.FromDate, req.ToDate}
	n := 4
	var filters []string
	var limit string

	if req.UserID.Valid {
		filters = append(filters, fmt.Sprintf(" AND user_id = $%d", n))
		values = append(values, req.UserID.V)
		n++
	}

	if req.ServiceName.Valid {
		filters = append(filters, fmt.Sprintf(" AND service_name = $%d", n))
		values = append(values, req.ServiceName.V)
		n++
	}

	if req.Limit > 0 {
		limit = fmt.Sprintf("LIMIT $%d", n)
		values = append(values, req.Limit)
		n++
	}

	rows, err := r.db().Query(ctx, fmt.Sprintf(sql, strings.Join(filters, ""), limit), values...)
	if err != nil {
		return nil, err
	}

	resp := []model.Subscription{}
	for rows.Next() {
		resp = append(resp, model.Subscription{})
		i := len(resp) - 1
		if err := rows.Scan(
			&resp[i].ID,
			&resp[i].ServiceName,
			&resp[i].Price,
			&resp[i].UserID,
			&resp[i].StartDate,
			&resp[i].EndDate,
			&resp[i].Created,
			&resp[i].Updated,
		); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *PGRepo) Update(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	var zero model.Subscription

	const sql = `UPDATE subscription 
	SET service_name=$2, price=$3, user_id=$4, start_date=$5, end_date=$6, updated=NOW()
	WHERE id = $1 AND updated = $7 AND deleted IS NULL
	RETURNING updated;`

	row := r.db().QueryRow(ctx, sql,
		req.ID,
		req.ServiceName,
		req.Price,
		req.UserID,
		req.StartDate,
		req.EndDate,
		req.Updated,
	)

	err := row.Scan(&req.Updated)

	if err != nil && err != pgx.ErrNoRows {
		return zero, err
	}

	if err == pgx.ErrNoRows {
		old, err := r.Get(ctx, req.ID)
		if err != nil {
			return zero, err
		}
		return old, fmt.Errorf("%w: optimistic locking conflict (2)", model.ErrConflict)
	}

	return req, nil
}

func (r *PGRepo) Delete(ctx context.Context, id int64) error {
	const sql = `UPDATE subscription SET deleted = NOW() WHERE id = $1 AND deleted IS NULL;`
	_, err := r.db().Exec(ctx, sql, id)
	return err
}

func (r *PGRepo) GetTotalCost(ctx context.Context, req model.SubscriptionFilter) (int64, error) {
	const sql = `
WITH filtered (price, actual_start, actual_end) AS NOT MATERIALIZED (
	SELECT 
		price,
		GREATEST(start_date, $1::date),
		LEAST(end_date, $2::date)
	FROM subscription
	WHERE start_date <= $2::date AND $1::date <= end_date %s
),
calculated_months (price, months) AS NOT MATERIALIZED (
	SELECT 
		price,
		(EXTRACT(YEAR FROM actual_end)::int - EXTRACT(YEAR FROM actual_start))::int * 12 +
		(EXTRACT(MONTH FROM actual_end)::int - EXTRACT(MONTH FROM actual_start))::int + 1
	FROM filtered
)
SELECT COALESCE(SUM(months * price), 0) AS total_cost 
FROM calculated_months;`

	values := []any{req.FromDate, req.ToDate}
	n := 3
	var filters []string

	if req.UserID.Valid {
		filters = append(filters, fmt.Sprintf(" AND user_id = $%d", n))
		values = append(values, req.UserID.V)
		n++
	}

	if req.ServiceName.Valid {
		filters = append(filters, fmt.Sprintf(" AND service_name = $%d", n))
		values = append(values, req.ServiceName.V)
		n++
	}

	var totalCost int64
	err := r.db().QueryRow(ctx, fmt.Sprintf(sql, strings.Join(filters, "")), values...).Scan(&totalCost)
	return totalCost, err
}

var _ service.Repository = &PGRepo{}
