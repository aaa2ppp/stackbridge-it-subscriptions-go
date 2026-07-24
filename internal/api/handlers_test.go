package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"subscriptions/internal/api"
	"subscriptions/internal/model"
	"testing"
	"time"

	"github.com/aaa2ppp/be"
	"github.com/google/uuid"
)

type mockService struct {
	create       func(req model.Subscription) (model.Subscription, error)
	delete       func(id int64) error
	get          func(id int64) (model.Subscription, error)
	getTotalCost func(req model.TotalCostRequest) (model.TotalCostResponse, error)
	list         func(req model.ListSubscriptionsRequest) ([]model.Subscription, error)
	update       func(req model.UpdateSubscriptionRequest) (model.Subscription, error)
	check        func(req any) bool
}

// Create implements [api.Service].
func (m *mockService) Create(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	if m.check != nil && !m.check(req) {
		return model.Subscription{}, model.ErrValidation
	}
	if m.create != nil {
		return m.create(req)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

// Delete implements [api.Service].
func (m *mockService) Delete(ctx context.Context, id int64) error {
	if m.check != nil && !m.check(id) {
		return model.ErrValidation
	}
	if m.delete != nil {
		return m.delete(id)
	}
	return model.ErrNotImplemented
}

// Get implements [api.Service].
func (m *mockService) Get(ctx context.Context, id int64) (model.Subscription, error) {
	if m.check != nil && !m.check(id) {
		return model.Subscription{}, model.ErrValidation
	}
	if m.get != nil {
		return m.get(id)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

// GetTotalCost implements [api.Service].
func (m *mockService) GetTotalCost(ctx context.Context, req model.TotalCostRequest) (model.TotalCostResponse, error) {
	if m.check != nil && !m.check(req) {
		return model.TotalCostResponse{}, model.ErrValidation
	}
	if m.getTotalCost != nil {
		return m.getTotalCost(req)
	}
	return model.TotalCostResponse{}, model.ErrNotImplemented
}

// List implements [api.Service].
func (m *mockService) List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
	if m.check != nil && !m.check(req) {
		return nil, model.ErrValidation
	}
	if m.list != nil {
		return m.list(req)
	}
	return nil, model.ErrNotImplemented
}

// Update implements [api.Service].
func (m *mockService) Update(ctx context.Context, req model.UpdateSubscriptionRequest) (model.Subscription, error) {
	if m.check != nil && !m.check(req) {
		return model.Subscription{}, model.ErrValidation
	}
	if m.update != nil {
		return m.update(req)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

var _ api.Service = &mockService{}

func MonthYear(m time.Month, y int) model.MonthYear {
	return model.MonthYear{Time: time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)}
}

func Date(y int, m time.Month, d int) model.Date {
	return model.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

func Nullable[T any](v T, defined, valid bool) model.Nullable[T] {
	return model.Nullable[T]{Value: v, Defined: defined, Valid: valid}
}

func TestHandlers(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		contentType    string
		body           string
		wantReq        any
		svc            mockService
		wantStatusCode int
		wantBody       string
	}{
		// --- Create ----------------------------------------------------------
		{
			name:        "create: success",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantReq: model.Subscription{
				ServiceName: "Yandex Plus",
				Price:       400,
				UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
				StartDate:   MonthYear(7, 2025),
				EndDate:     Nullable(MonthYear(12, 2025), true, true),
			},
			svc: mockService{
				create: func(req model.Subscription) (model.Subscription, error) { req.ID = 42; return req, nil },
			},
			wantStatusCode: 201,
			wantBody: `{
				"id":           42,
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0001-01-01T00:00:00Z"
			}`,
		},
		// --- List ------------------------------------------------------------
		{
			"list: success",
			"GET /subscriptions",
			"",
			``,
			model.ListSubscriptionsRequest{
				AfterID: 0,
				Limit:   1,
			},
			mockService{
				list: func(req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
					return []model.Subscription{
						{
							ID:          42,
							ServiceName: "Yandex Plus",
							Price:       400,
							UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
							StartDate:   MonthYear(7, 2025),
							EndDate:     Nullable(MonthYear(12, 2025), true, true),
						},
					}, nil
				},
			},
			200,
			`[
				{
					"id":           42,
					"service_name": "Yandex Plus",
					"price":        400,
					"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
					"start_date":   "07-2025",
					"end_date":     "12-2025",
					"created":      "0001-01-01T00:00:00Z",
					"updated":      "0001-01-01T00:00:00Z"
				}
			]`,
		},
		// --- Get ------------------------------------------------------------
		{
			"get: success",
			"GET /subscriptions/42",
			"",
			``,
			int64(42),
			mockService{
				get: func(id int64) (model.Subscription, error) {
					return model.Subscription{
						ID:          42,
						ServiceName: "Yandex Plus",
						Price:       400,
						UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
						StartDate:   MonthYear(7, 2025),
						EndDate:     Nullable(MonthYear(12, 2025), true, true),
					}, nil
				},
			},
			200,
			`{
				"id":           42,
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0001-01-01T00:00:00Z"
			}`,
		},
		// --- Update -----------------------------------------------------------
		{
			"update: success",
			"PATCH /subscriptions/42",
			"application/json",
			`{
				"price":    400,
				"end_date": null
			}`,
			model.UpdateSubscriptionRequest{
				ID:      int64(42),
				Price:   Nullable(int64(400), true, true),
				EndDate: Nullable(model.MonthYear{}, true, false),
			},
			mockService{
				update: func(req model.UpdateSubscriptionRequest) (model.Subscription, error) {
					return model.Subscription{
						ID:          int64(42),
						ServiceName: "Yandex Plus",
						Price:       400,
						UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
						StartDate:   MonthYear(7, 2025),
					}, nil
				},
			},
			200,
			`{
				"id":           42,
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0001-01-01T00:00:00Z"
			}`,
		},
		// --- Delete ----------------------------------------------------------
		{
			"delete: success",
			"DELETE /subscriptions/42",
			"",
			``,
			int64(42),
			mockService{
				delete: func(id int64) error { return nil },
			},
			204,
			"",
		},
		{
			"delete: not found",
			"DELETE /subscriptions/42",
			"",
			``,
			int64(42),
			mockService{
				delete: func(id int64) error { return model.ErrNotFound },
			},
			204,
			``,
		},
		{
			"delete: bad id",
			"DELETE /subscriptions/forty-two",
			"",
			"",
			``,
			mockService{
				delete: func(id int64) error { return model.ErrNotFound },
			},
			400,
			``,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.svc
			if tt.wantReq != nil {
				svc.check = func(req any) bool {
					return be.Equal(be.Diff(t), req, tt.wantReq)
				}
			}

			server := httptest.NewServer(api.New(&svc))
			defer server.Close()

			method, url, _ := strings.Cut(tt.query, " ")
			httpReq, err := http.NewRequest(method, server.URL+url, strings.NewReader(tt.body))
			be.Err(t, err, nil)

			if tt.contentType != "" {
				httpReq.Header.Set("content-type", tt.contentType)
			}

			httpResp, err := http.DefaultClient.Do(httpReq)
			be.Err(t, err, nil)
			defer httpResp.Body.Close()

			if !be.Equal(t, httpResp.StatusCode, tt.wantStatusCode) {
				body, _ := io.ReadAll(httpResp.Body)
				if len(body) > 256 {
					body = append(body[:253], "..."...)
				}
				t.Logf("resp:\n%s", body)
				return
			}

			if tt.wantBody != "" {
				var wantResp any
				err := json.NewDecoder(strings.NewReader(tt.wantBody)).Decode(&wantResp)
				be.Err(t, err, nil)

				var resp any
				err = json.NewDecoder(httpResp.Body).Decode(&resp)
				be.Err(t, err, nil)

				be.Equal(be.Diff(t), resp, wantResp)
			}
		})
	}
}
