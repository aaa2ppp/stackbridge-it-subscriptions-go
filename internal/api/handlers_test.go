package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	getTotalCost func(req model.SubscriptionFilter) (model.TotalCostResponse, error)
	list         func(req model.ListSubscriptionsRequest) ([]model.Subscription, error)
	update       func(req model.UpdateSubscriptionRequest) (model.Subscription, error)
	gotReq       any
}

// Create implements [api.Service].
func (m *mockService) Create(ctx context.Context, req model.Subscription) (model.Subscription, error) {
	m.gotReq = req
	if m.create != nil {
		return m.create(req)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

// Delete implements [api.Service].
func (m *mockService) Delete(ctx context.Context, id int64) error {
	m.gotReq = id
	if m.delete != nil {
		return m.delete(id)
	}
	return model.ErrNotImplemented
}

// Get implements [api.Service].
func (m *mockService) Get(ctx context.Context, id int64) (model.Subscription, error) {
	m.gotReq = id
	if m.get != nil {
		return m.get(id)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

// GetTotalCost implements [api.Service].
func (m *mockService) GetTotalCost(ctx context.Context, req model.SubscriptionFilter) (model.TotalCostResponse, error) {
	m.gotReq = req
	if m.getTotalCost != nil {
		return m.getTotalCost(req)
	}
	return model.TotalCostResponse{}, model.ErrNotImplemented
}

// List implements [api.Service].
func (m *mockService) List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
	m.gotReq = req
	if m.list != nil {
		return m.list(req)
	}
	return nil, model.ErrNotImplemented
}

// Update implements [api.Service].
func (m *mockService) Update(ctx context.Context, req model.UpdateSubscriptionRequest) (model.Subscription, error) {
	m.gotReq = req
	if m.update != nil {
		return m.update(req)
	}
	return model.Subscription{}, model.ErrNotImplemented
}

var _ api.Service = &mockService{}

func MonthYear(m time.Month, y int) model.MonthYear {
	return model.MonthYear{Time: time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)}
}

func CurrentMonthYear() model.MonthYear {
	now := time.Now()
	return MonthYear(now.Month(), now.Year())
}

func Nullable[T any](v T, valid, defined bool) model.Nullable[T] {
	return model.Nullable[T]{Null: sql.Null[T]{V: v, Valid: valid}, Defined: defined}
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
			name:        "create success",
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
				EndDate:     MonthYear(12, 2025),
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
		{
			name:        "create null end_date",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     null
			}`,
			wantReq: model.Subscription{
				ServiceName: "Yandex Plus",
				Price:       400,
				UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
				StartDate:   MonthYear(7, 2025),
				EndDate:     api.MonthYearInfinity,
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
				"end_date":     "12-9999",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0001-01-01T00:00:00Z"
			}`,
		},
		{
			name:        "create missing service_name",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "create invalid price value",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        -1,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "create missing user_id",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "create missing start_date",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "create invalid date range",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "12-2025",
				"end_date":     "07-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "create service unknown error",
			query:       "POST /subscriptions",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantReq: true,
			svc: mockService{
				create: func(req model.Subscription) (model.Subscription, error) {
					return model.Subscription{}, errors.New("unknown error")
				},
			},
			wantStatusCode: 500,
		},
		// --- List ------------------------------------------------------------
		{
			name:        "list success",
			query:       "GET /subscriptions?from_date=07-2025&to_date=08-2025",
			contentType: "",
			wantReq: model.ListSubscriptionsRequest{
				AfterID: 0,
				Limit:   api.DefaultLimit,
				SubscriptionFilter: model.SubscriptionFilter{
					FromDate: MonthYear(7, 2025),
					ToDate:   MonthYear(8, 2025),
				},
			},
			svc: mockService{
				list: func(req model.ListSubscriptionsRequest) ([]model.Subscription, error) {
					return []model.Subscription{
						{
							ID:          42,
							ServiceName: "Yandex Plus",
							Price:       400,
							UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
							StartDate:   MonthYear(7, 2025),
							EndDate:     MonthYear(12, 2025),
						},
					}, nil
				},
			},
			wantStatusCode: 200,
			wantBody: `[
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
		{
			name:           "list bad query",
			query:          "GET /subscriptions?after_id=invalid",
			wantStatusCode: 400,
		},
		{
			name:           "list invalid after_id value",
			query:          "GET /subscriptions?after_id=-1",
			wantStatusCode: 400,
		},
		{
			name:           "list invalid limit value",
			query:          "GET /subscriptions?limit=0",
			wantStatusCode: 400,
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
						EndDate:     MonthYear(12, 2025),
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
		{
			name:           "get bad id",
			query:          "GET /subscriptions/forty-two",
			wantStatusCode: 400,
		},
		{
			name:    "get not found",
			query:   "GET /subscriptions/42",
			wantReq: int64(42),
			svc: mockService{
				get: func(id int64) (model.Subscription, error) { return model.Subscription{}, model.ErrNotFound },
			},
			wantStatusCode: 404,
		},
		// --- Update -----------------------------------------------------------
		{
			name:        "update success",
			query:       "PATCH /subscriptions/42",
			contentType: "application/json",
			body: `{
				"price":    400,
				"end_date": null
			}`,
			wantReq: model.UpdateSubscriptionRequest{
				ID:      int64(42),
				Price:   Nullable(int64(400), true, true),
				EndDate: Nullable(api.MonthYearInfinity, true, true), // null -> 12-9999
			},
			svc: mockService{
				update: func(req model.UpdateSubscriptionRequest) (model.Subscription, error) {
					return model.Subscription{
						ID:          int64(42),
						ServiceName: "Yandex Plus",
						Price:       400,
						UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
						StartDate:   MonthYear(7, 2025),
						EndDate:     api.MonthYearInfinity,
					}, nil
				},
			},
			wantStatusCode: 200,
			wantBody: `{
				"id":           42,
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-9999",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0001-01-01T00:00:00Z"
			}`,
		},
		{
			name:           "update bad id",
			query:          "PATCH /subscriptions/forty-two",
			body:           `{"price":400}`,
			contentType:    "application/json",
			wantStatusCode: 400,
		},
		{
			name:           "update bad json",
			query:          "PATCH /subscriptions/42",
			body:           `invalid`,
			contentType:    "application/json",
			wantStatusCode: 400,
		},
		{
			name:        "update invalid service_name",
			query:       "PATCH /subscriptions/42",
			contentType: "application/json",
			body: `{
				"service_name": "",
				"price":        400,
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "update invalid price value",
			query:       "PATCH /subscriptions/42",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        -1,
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantStatusCode: 400,
		},
		{
			name:        "update invalid date range",
			query:       "PATCH /subscriptions/42",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"start_date":   "12-2025",
				"end_date":     "07-2025"
			}`,
			svc:            mockService{},
			wantStatusCode: 400,
		},
		{
			name:        "update service unknown error",
			query:       "PATCH /subscriptions/42",
			contentType: "application/json",
			body: `{
				"service_name": "Yandex Plus",
				"price":        400,
				"start_date":   "07-2025",
				"end_date":     "12-2025"
			}`,
			wantReq: true,
			svc: mockService{
				update: func(req model.UpdateSubscriptionRequest) (model.Subscription, error) {
					return model.Subscription{}, errors.New("unknown error")
				},
			},
			wantStatusCode: 500,
		},
		{
			name:        "update not found",
			query:       "PATCH /subscriptions/42",
			body:        `{"price":400}`,
			contentType: "application/json",
			wantReq:     true,
			svc: mockService{
				update: func(req model.UpdateSubscriptionRequest) (model.Subscription, error) {
					return model.Subscription{}, model.ErrNotFound
				},
			},
			wantStatusCode: 404,
		},
		{
			name:        "update conflict",
			query:       "PATCH /subscriptions/42",
			body:        `{"price":400}`,
			contentType: "application/json",
			wantReq:     true,
			svc: mockService{
				update: func(req model.UpdateSubscriptionRequest) (model.Subscription, error) {
					return model.Subscription{
						ID:          int64(42),
						ServiceName: "Yandex Plus",
						Price:       400,
						UserID:      uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"),
						StartDate:   MonthYear(7, 2025),
						EndDate:     MonthYear(12, 2025),
						Updated:     time.Date(2, 2, 2, 2, 2, 2, 0, time.UTC),
					}, model.ErrConflict
				},
			},
			wantStatusCode: 409,
			wantBody: fmt.Sprintf(`{
			"current": {
				"id":           42,
				"service_name": "Yandex Plus",
				"price":        400,
				"user_id":      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date":   "07-2025",
				"end_date":     "12-2025",
				"created":      "0001-01-01T00:00:00Z",
				"updated":      "0002-02-02T02:02:02Z"
			},
			"error": %q
			}`, model.ErrConflict),
		},
		// --- Delete ----------------------------------------------------------
		{
			name:    "delete success",
			query:   "DELETE /subscriptions/42",
			wantReq: int64(42),
			svc: mockService{
				delete: func(id int64) error { return nil },
			},
			wantStatusCode: 204,
		},
		{
			name:    "delete not found",
			query:   "DELETE /subscriptions/42",
			wantReq: int64(42),
			svc: mockService{
				delete: func(id int64) error { return model.ErrNotFound },
			},
			wantStatusCode: 204,
		},
		{
			name:           "delete bad id",
			query:          "DELETE /subscriptions/forty-two",
			wantStatusCode: 400,
		},
		// --- GetTotalCost ----------------------------------------------------
		{
			name:  "total success",
			query: "GET /subscriptions/total?from_date=07-2025&to_date=08-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Yandex%20Plus",
			wantReq: model.SubscriptionFilter{
				UserID:      Nullable(uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba"), true, true),
				ServiceName: Nullable("Yandex Plus", true, true),
				FromDate:    MonthYear(7, 2025),
				ToDate:      MonthYear(8, 2025),
			},
			svc: mockService{
				getTotalCost: func(req model.SubscriptionFilter) (model.TotalCostResponse, error) {
					return model.TotalCostResponse{TotalCost: 1000}, nil
				},
			},
			wantStatusCode: 200,
			wantBody:       `{"total_cost":1000}`,
		},
		{
			name:  "total enpty query",
			query: "GET /subscriptions/total",
			wantReq: model.SubscriptionFilter{
				FromDate: MonthYear(1, 1),
				ToDate:   CurrentMonthYear(),
			},
			svc: mockService{
				getTotalCost: func(req model.SubscriptionFilter) (model.TotalCostResponse, error) {
					return model.TotalCostResponse{}, nil
				},
			},
			wantStatusCode: 200,
		},
		{
			name:           "total invalid service_name",
			query:          "GET /subscriptions/total?service_name=",
			wantStatusCode: 400,
		},
		{
			name:           "total invalid user_id",
			query:          "GET /subscriptions/total?user_id=invalid",
			wantStatusCode: 400,
		},
		{
			name:           "total invalid date range",
			query:          "GET /subscriptions/total?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&from_date=08-2025&to_date=07-2025",
			wantStatusCode: 400,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.svc

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

			if wantReq, ok := tt.wantReq.(bool); ok {
				be.Equal(t, (svc.gotReq != nil), wantReq)
			} else {
				be.Equal(be.Diff(t), svc.gotReq, tt.wantReq)
			}

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

				var gotResp any
				err = json.NewDecoder(httpResp.Body).Decode(&gotResp)
				be.Err(t, err, nil)

				be.Equal(be.Diff(t), gotResp, wantResp)
			}
		})
	}
}
