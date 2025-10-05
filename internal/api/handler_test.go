package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subscription-service/internal/api"
	"subscription-service/internal/model"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) CreateSubscription(ctx context.Context, in service.CreateInput) (*model.Subscription, error) {
	args := m.Called(ctx, in)
	if sub, ok := args.Get(0).(*model.Subscription); ok {
		return sub, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockService) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	args := m.Called(ctx, id)
	if sub, ok := args.Get(0).(*model.Subscription); ok {
		return sub, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockService) UpdateSubscription(ctx context.Context, id string, in service.UpdateInput) (*model.Subscription, error) {
	args := m.Called(ctx, id, in)
	if sub, ok := args.Get(0).(*model.Subscription); ok {
		return sub, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockService) DeleteSubscription(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockService) ListSubscriptions(ctx context.Context, filter repository.ListFilter) ([]*model.Subscription, error) {
	args := m.Called(ctx, filter)
	if subs, ok := args.Get(0).([]*model.Subscription); ok {
		return subs, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockService) SumForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error) {
	args := m.Called(ctx, from, to, userID, serviceName)
	return args.Get(0).(int64), args.Error(1)
}

func TestCreateSubscription_Success(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	userID := uuid.New().String()
	body := map[string]any{
		"service_name": "Netflix",
		"price":        499,
		"user_id":      userID,
		"start_date":   "10-2025",
	}
	b, _ := json.Marshal(body)

	start, _ := time.Parse("01-2006", "10-2025")
	end := start.AddDate(0, 0, 30)
	created := &model.Subscription{
		ID:          uuid.New().String(),
		ServiceName: "Netflix",
		Price:       499,
		UserID:      userID,
		StartDate:   start,
		EndDate:     &end,
	}

	svc.On("CreateSubscription", mock.Anything, mock.AnythingOfType("service.CreateInput")).Return(created, nil)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(b))
	w := httptest.NewRecorder()
	h.CreateSubscription(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var got model.Subscription
	_ = json.NewDecoder(resp.Body).Decode(&got)
	assert.Equal(t, created.ID, got.ID)
	svc.AssertExpectations(t)
}

func TestCreateSubscription_InvalidJSON(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString("{bad json"))
	w := httptest.NewRecorder()
	h.CreateSubscription(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSubscriptionByID_NotFound(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	id := uuid.New().String()
	svc.On("GetByID", mock.Anything, id).Return(nil, repository.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+id, nil)
	req = muxWithParam(req, "id", id)
	w := httptest.NewRecorder()
	h.GetSubscriptionByID(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteSubscription_Success(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	id := uuid.New().String()
	sub := &model.Subscription{
		ID:          id,
		UserID:      uuid.New().String(),
		ServiceName: "YouTube",
	}
	svc.On("GetByID", mock.Anything, id).Return(sub, nil)
	svc.On("DeleteSubscription", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/subscriptions/"+id, nil)
	req = muxWithParam(req, "id", id)
	w := httptest.NewRecorder()
	h.DeleteSubscription(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestGetTotalCost_Success(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	from := "2025-01-01"
	to := "2025-12-31"
	svc.On("SumForPeriod", mock.Anything, mock.Anything, mock.Anything, (*string)(nil), (*string)(nil)).Return(int64(999), nil)

	req := httptest.NewRequest(http.MethodGet, "/total-cost?from="+from+"&to="+to, nil)
	w := httptest.NewRecorder()
	h.GetTotalCost(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]int64
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, int64(999), result["total"])
}

func TestGetTotalCost_MissingParams(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/total-cost", nil)
	w := httptest.NewRecorder()
	h.GetTotalCost(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func muxWithParam(r *http.Request, key, val string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func TestListSubscriptions_Success(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	now := time.Now()
	subs := []*model.Subscription{
		{
			ID:          uuid.New().String(),
			ServiceName: "Netflix",
			Price:       499,
			UserID:      uuid.New().String(),
			StartDate:   now,
			EndDate:     func() *time.Time { e := now.AddDate(0, 0, 30); return &e }(),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			ServiceName: "Spotify",
			Price:       299,
			UserID:      uuid.New().String(),
			StartDate:   now,
			EndDate:     func() *time.Time { e := now.AddDate(0, 0, 30); return &e }(),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	filter := repository.ListFilter{Limit: 50, Offset: 0}
	svc.On("ListSubscriptions", mock.Anything, filter).Return(subs, nil)

	r := chi.NewRouter()
	r.Get("/subscriptions", h.ListSubscriptions)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got []*model.Subscription
	_ = json.NewDecoder(resp.Body).Decode(&got)
	assert.Len(t, got, 2)
	assert.Equal(t, "Netflix", got[0].ServiceName)
	svc.AssertExpectations(t)
}

func TestUpdateSubscription_Success(t *testing.T) {
	svc := new(mockService)
	h := api.NewHandler(svc)

	id := uuid.New().String()
	userID := uuid.New().String()

	body := map[string]any{
		"service_name": "Spotify",
		"price":        299,
		"user_id":      userID,
		"start_date":   "09-2025",
	}
	b, _ := json.Marshal(body)

	start, _ := time.Parse("01-2006", "09-2025")
	end := start.AddDate(0, 0, 30)

	updated := &model.Subscription{
		ID:          id,
		ServiceName: "Spotify",
		Price:       299,
		UserID:      userID,
		StartDate:   start,
		EndDate:     &end,
	}

	svc.On("UpdateSubscription", mock.Anything, id, mock.AnythingOfType("service.UpdateInput")).Return(updated, nil)

	r := chi.NewRouter()
	r.Put("/subscriptions/{id}", h.UpdateSubscription)

	req := httptest.NewRequest(http.MethodPut, "/subscriptions/"+id, bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var got model.Subscription
	_ = json.NewDecoder(resp.Body).Decode(&got)
	assert.Equal(t, "Spotify", got.ServiceName)
	svc.AssertExpectations(t)
}
