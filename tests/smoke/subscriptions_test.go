package smoke

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"subscriptions/internal/api"
	"subscriptions/internal/config"
	"subscriptions/internal/lib/logging"
	"subscriptions/internal/model"
	"subscriptions/internal/repo/pgrepo"
	"subscriptions/internal/service"

	"github.com/aaa2ppp/be"
	"github.com/aaa2ppp/be/tb"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

const migrationsPath = "../../migrations"

// Smoke test проверяет базовую работоспособность:
//   - поднимает БД в контейнере
//   - накатывает миграции
//   - создает подписку
//   - получает ее
//   - обновляем
//   - считает total cost
//   - удаляем
//   - проверяем, что удалили
func TestSmoke(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// Поднимаем контейнер
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.4-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "subscriptions",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	be.Err(t, err, nil)
	defer postgresC.Terminate(ctx) //nolint:errcheck

	// Подключаемся
	host, _ := postgresC.Host(ctx)
	port, _ := postgresC.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=subscriptions sslmode=disable",
		host, port.Port())

	t.Logf("DSN: %q", dsn)
	sqlDB, err := sql.Open("postgres", dsn)
	be.Err(t, err, nil)

	be.Err(t, goose.SetDialect("postgres"), nil)
	err = goose.DownTo(sqlDB, migrationsPath, 0)
	be.Err(t, err, nil)
	err = goose.Up(sqlDB, migrationsPath)
	be.Err(t, err, nil)

	cfg := config.DB{
		Addr:     fmt.Sprintf("%s:%s", host, port.Port()),
		User:     "postgres",
		Password: "postgres",
		DBName:   "subscriptions",
		SSLMode:  "disable",
	}

	t.Logf("cfg: %+v", cfg)

	repo, _ := pgrepo.Open(ctx, cfg)
	defer repo.Close()

	svc := service.New(repo)
	apiHandler := logging.HTTPLogging(logger, api.New(svc))
	server := httptest.NewServer(apiHandler)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	userID := [2]uuid.UUID{uuid.New(), uuid.New()}

	var created model.Subscription

	if !t.Run("Create", func(t *testing.T) {
		createBody := map[string]any{
			"service_name": "Service0",
			"price":        100,
			"user_id":      userID[0],
			"start_date":   "07-2025",
		}
		b, _ := json.Marshal(createBody)

		resp, err := client.Post(server.URL+"/subscriptions", "application/json", bytes.NewReader(b))
		be.Err(t, err, nil)
		defer resp.Body.Close() //nolint:errcheck

		be.Equal(t, resp.StatusCode, 201)
		be.Err(t, json.NewDecoder(resp.Body).Decode(&created), nil)

		be.True(t, created.ID > 0)
		be.Equal(t, created.Price, 100)
		be.Equal(t, created.StartDate, model.MonthYear{Time: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)})
		be.Equal(t, created.EndDate, api.MonthYearInfinity) // бессрочная
	}) {
		return
	}

	if !t.Run("Get", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/subscriptions/%d", server.URL, created.ID))
		be.Err(t, err, nil)
		defer resp.Body.Close() //nolint:errcheck

		be.Equal(t, resp.StatusCode, 200)
		var sub model.Subscription
		be.Err(t, json.NewDecoder(resp.Body).Decode(&sub), nil)

		be.Equal(tb.Diff(t), sub, created)
	}) {
		return
	}

	if !t.Run("Update", func(t *testing.T) {
		updateBody := map[string]any{
			"price":   150,
			"updated": created.Updated,
		}
		b, _ := json.Marshal(updateBody)

		httpReq, err := http.NewRequest("PATCH", fmt.Sprintf("%s/subscriptions/%d", server.URL, created.ID), bytes.NewReader(b))
		httpReq.Header.Set("content-type", "application/json")
		be.Err(t, err, nil)
		resp, err := client.Do(httpReq)
		be.Err(t, err, nil)
		defer resp.Body.Close() //nolint:errcheck

		be.Equal(t, resp.StatusCode, 200)
		var sub model.Subscription
		be.Err(t, json.NewDecoder(resp.Body).Decode(&sub), nil)

		be.Equal(t, sub.Price, 150)
	}) {
		return
	}

	if !t.Run("Create2", func(t *testing.T) {
		createBody := map[string]any{
			"service_name": "Service1",
			"price":        100,
			"user_id":      userID[1],
			"start_date":   "07-2025",
		}
		b, _ := json.Marshal(createBody)

		resp, err := client.Post(server.URL+"/subscriptions", "application/json", bytes.NewReader(b))
		be.Err(t, err, nil)
		defer resp.Body.Close() //nolint:errcheck

		be.Equal(t, resp.StatusCode, 201)
		be.Err(t, json.NewDecoder(resp.Body).Decode(&created), nil)

		be.True(t, created.ID > 0)
		be.Equal(t, created.Price, 100)
		be.Equal(t, created.StartDate, model.MonthYear{Time: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)})
		be.Equal(t, created.EndDate, api.MonthYearInfinity) // бессрочная
	}) {
		return
	}

	if !t.Run("Total cost", func(t *testing.T) {
		httpResp, err := client.Get(fmt.Sprintf("%s/subscriptions/total?from_date=08-2025&to_date=10-2025", server.URL))
		be.Err(t, err, nil)
		defer httpResp.Body.Close() //nolint:errcheck

		be.Equal(t, httpResp.StatusCode, 200)
		var resp model.TotalCostResponse
		be.Err(t, json.NewDecoder(httpResp.Body).Decode(&resp), nil)
		be.Equal(t, resp.TotalCost, 450+300)
	}) {
		return
	}

	if !t.Run("Total cost user0", func(t *testing.T) {
		httpResp, err := client.Get(fmt.Sprintf("%s/subscriptions/total?from_date=06-2025&to_date=09-2025&user_id=%v", server.URL, userID[0]))
		be.Err(t, err, nil)
		defer httpResp.Body.Close() //nolint:errcheck

		be.Equal(t, httpResp.StatusCode, 200)
		var resp model.TotalCostResponse
		be.Err(t, json.NewDecoder(httpResp.Body).Decode(&resp), nil)
		be.Equal(t, resp.TotalCost, 450)
	}) {
		return
	}

	if !t.Run("Total cost service1", func(t *testing.T) {
		httpResp, err := client.Get(fmt.Sprintf("%s/subscriptions/total?from_date=06-2025&to_date=09-2025&service_name=Service1", server.URL))
		be.Err(t, err, nil)
		defer httpResp.Body.Close() //nolint:errcheck

		be.Equal(t, httpResp.StatusCode, 200)
		var resp model.TotalCostResponse
		be.Err(t, json.NewDecoder(httpResp.Body).Decode(&resp), nil)
		be.Equal(t, resp.TotalCost, 300)
	}) {
		return
	}

	if !t.Run("Delete", func(t *testing.T) {
		httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/subscriptions/%d", server.URL, created.ID), nil)
		be.Err(t, err, nil)
		httpResp, err := client.Do(httpReq)
		be.Err(t, err, nil)
		defer httpResp.Body.Close() //nolint:errcheck
		be.Equal(t, httpResp.StatusCode, 204)
	}) {
		return
	}

	if !t.Run("Check what deleted", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/subscriptions/%d", server.URL, created.ID))
		be.Err(t, err, nil)
		defer resp.Body.Close() //nolint:errcheck
		be.Equal(t, resp.StatusCode, 404)
	}) {
		return
	}
}
