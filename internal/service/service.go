package service

import (
	"context"
	"subscriptions/internal/api"
	"subscriptions/internal/model"
)

type Repository interface {
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Create implements [api.Service].
func (s *Service) Create(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	return model.Subscription{}, model.ErrNotImplemented
}

// Delete implements [api.Service].
func (s *Service) Delete(ctx context.Context, id int64) error {
	return model.ErrNotImplemented
}

// Get implements [api.Service].
func (s *Service) Get(ctx context.Context, id int64) (model.Subscription, error) {
	return model.Subscription{}, model.ErrNotImplemented
}

// List implements [api.Service].
func (s *Service) List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
	return nil, model.ErrNotImplemented
}

// Update implements [api.Service].
func (s *Service) Update(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	return model.Subscription{}, model.ErrNotImplemented
}

// GetTotalCost implements [api.Service].
func (s *Service) GetTotalCost(ctx context.Context, req model.GetTotalCostRequest) (model.GetTotalCostResponse, error) {
	panic("unimplemented")
}

var _ api.Service = &Service{}
