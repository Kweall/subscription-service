package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"subscription-service/internal/model"
)

var ErrNotFound = errors.New("not found")

type ListFilter struct {
	UserID      *string
	ServiceName *string
	Limit       int
	Offset      int
}

type SubscriptionRepo interface {
	Create(ctx context.Context, s *model.Subscription) error
	GetByID(ctx context.Context, id string) (*model.Subscription, error)
	Update(ctx context.Context, s *model.Subscription) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*model.Subscription, error)
	TotalCostForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error)
}

type pgRepo struct {
	db *sql.DB
}

func NewPGRepo(db *sql.DB) SubscriptionRepo {
	return &pgRepo{db: db}
}

func (p *pgRepo) Create(ctx context.Context, s *model.Subscription) error {
	query := `INSERT INTO subscriptions
      (id, service_name, price, user_id, start_date, end_date, created_at, updated_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := p.db.ExecContext(ctx, query,
		s.ID, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate, s.CreatedAt, s.UpdatedAt)
	return err
}

func (p *pgRepo) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	q := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
          FROM subscriptions WHERE id = $1`
	row := p.db.QueryRowContext(ctx, q, id)
	s := &model.Subscription{}
	var end sql.NullTime
	if err := row.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &end, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if end.Valid {
		s.EndDate = &end.Time
	}
	return s, nil
}

func (p *pgRepo) Update(ctx context.Context, s *model.Subscription) error {
	q := `UPDATE subscriptions SET service_name=$1, price=$2, user_id=$3, start_date=$4, end_date=$5, updated_at=$6
          WHERE id=$7`
	res, err := p.db.ExecContext(ctx, q, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate, s.UpdatedAt, s.ID)
	if err != nil {
		return err
	}
	if ra, _ := res.RowsAffected(); ra == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *pgRepo) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM subscriptions WHERE id = $1`
	res, err := p.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	if ra, _ := res.RowsAffected(); ra == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *pgRepo) List(ctx context.Context, filter ListFilter) ([]*model.Subscription, error) {
	q := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
          FROM subscriptions
          WHERE ($1::uuid IS NULL OR user_id = $1::uuid)
            AND ($2::text IS NULL OR service_name = $2::text)
          ORDER BY created_at DESC
          LIMIT $3 OFFSET $4`

	var uid, sname interface{}
	if filter.UserID != nil {
		uid = *filter.UserID
	}
	if filter.ServiceName != nil {
		sname = *filter.ServiceName
	}

	rows, err := p.db.QueryContext(ctx, q, uid, sname, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Subscription
	for rows.Next() {
		s := &model.Subscription{}
		var end sql.NullTime
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &end, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if end.Valid {
			s.EndDate = &end.Time
		}
		out = append(out, s)
	}
	return out, nil
}

func (p *pgRepo) TotalCostForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error) {
	q := `SELECT COALESCE(SUM(price),0)
          FROM subscriptions
          WHERE start_date <= $1
            AND end_date >= $2
            AND ($3::uuid IS NULL OR user_id = $3::uuid)
            AND ($4::text IS NULL OR service_name = $4::text)`

	var uid, sname interface{}
	if userID != nil {
		uid = *userID
	}
	if serviceName != nil {
		sname = *serviceName
	}

	var total int64
	err := p.db.QueryRowContext(ctx, q, to, from, uid, sname).Scan(&total)
	return total, err
}
