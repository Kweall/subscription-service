package service

import (
	"context"
	"errors"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var ErrInvalid = errors.New("invalid input")

type SubscriptionService interface {
	CreateSubscription(ctx context.Context, in CreateInput) (*model.Subscription, error)
	GetByID(ctx context.Context, id string) (*model.Subscription, error)
	UpdateSubscription(ctx context.Context, id string, in UpdateInput) (*model.Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
	ListSubscriptions(ctx context.Context, filter repository.ListFilter) ([]*model.Subscription, error)
	SumForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error)
}

type serviceImpl struct {
	repo repository.SubscriptionRepo
}

func NewSubscriptionService(r repository.SubscriptionRepo) SubscriptionService {
	return &serviceImpl{repo: r}
}

type CreateInput struct {
	ServiceName string
	Price       int
	UserID      string
	StartDate   time.Time
	EndDate     *time.Time
}

type UpdateInput struct {
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      string     `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

func (s *serviceImpl) CreateSubscription(ctx context.Context, in CreateInput) (*model.Subscription, error) {
	if in.ServiceName == "" || in.Price < 0 {
		return nil, ErrInvalid
	}
	if _, err := uuid.Parse(in.UserID); err != nil {
		return nil, ErrInvalid
	}

	start := in.StartDate
	var end time.Time
	if in.EndDate != nil {
		end = *in.EndDate
		if start.After(end) {
			return nil, ErrInvalid
		}
	} else {
		end = start.AddDate(0, 0, 30)
	}

	now := time.Now().UTC()
	id := uuid.New().String()
	sub := &model.Subscription{
		ID:          id,
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   start,
		EndDate:     &end,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if sub.StartDate.IsZero() {
		sub.StartDate = now
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		log.Error().Err(err).Msg("repo.Create failed")
		return nil, err
	}
	return sub, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *serviceImpl) UpdateSubscription(ctx context.Context, id string, in UpdateInput) (*model.Subscription, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.EndDate != nil && in.EndDate.Before(in.StartDate) {
		return nil, ErrInvalid
	}
	existing.ServiceName = in.ServiceName
	existing.Price = in.Price
	existing.UserID = in.UserID
	existing.StartDate = in.StartDate
	if in.EndDate == nil {
		end := in.StartDate.AddDate(0, 0, 30)
		existing.EndDate = &end
	} else {
		existing.EndDate = in.EndDate
	}
	existing.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *serviceImpl) DeleteSubscription(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *serviceImpl) ListSubscriptions(ctx context.Context, filter repository.ListFilter) ([]*model.Subscription, error) {
	return s.repo.List(ctx, filter)
}

func (s *serviceImpl) SumForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error) {
	if to.Before(from) {
		return 0, ErrInvalid
	}
	return s.repo.TotalCostForPeriod(ctx, from, to, userID, serviceName)
}
