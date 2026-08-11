package service

import (
	"context"
	"fmt"

	"aaa2ppp/subscriptions/internal/api"
	"aaa2ppp/subscriptions/internal/model"
)

type Repository interface {
	Create(ctx context.Context, req model.Subscription) (model.Subscription, error)
	Get(ctx context.Context, id int64) (model.Subscription, error)
	List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error)
	Update(ctx context.Context, req model.Subscription) (model.Subscription, error)
	Delete(ctx context.Context, id int64) error
	GetTotalCost(ctx context.Context, req model.SubscriptionFilter) (int64, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create implements [api.Service].
func (s *Service) Create(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	return s.repo.Create(ctx, req)
}

// Delete implements [api.Service].
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Get implements [api.Service].
func (s *Service) Get(ctx context.Context, id int64) (model.Subscription, error) {
	return s.repo.Get(ctx, id)
}

// List implements [api.Service].
func (s *Service) List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
	return s.repo.List(ctx, req)
}

// Update implements [api.Service].
func (s *Service) Update(ctx context.Context, req model.UpdateSubscriptionRequest) (model.Subscription, error) {
	var zero model.Subscription

	old, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return zero, err
	}
	if !old.Updated.Equal(req.Updated) {
		return old, fmt.Errorf("%w: optimistic locking conflict (1)", model.ErrConflict)
	}

	sub := old
	if req.ServiceName.Defined {
		sub.ServiceName = req.ServiceName.V
	}
	if req.Price.Defined {
		sub.Price = req.Price.V
	}
	if req.StartDate.Defined {
		sub.StartDate = req.StartDate.V
	}
	if req.EndDate.Defined {
		sub.EndDate = req.EndDate.V
	}
	if sub.EndDate.Before(sub.StartDate.Time) {
		return old, fmt.Errorf("%w: end_date cannot be before start_date", model.ErrConflict)
	}

	return s.repo.Update(ctx, sub)
}

// GetTotalCost implements [api.Service].
func (s *Service) GetTotalCost(ctx context.Context, req model.SubscriptionFilter) (model.TotalCostResponse, error) {
	totalCost, err := s.repo.GetTotalCost(ctx, req)
	return model.TotalCostResponse{TotalCost: totalCost}, err
}

var _ api.Service = &Service{}
