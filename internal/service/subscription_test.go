package service_test

import (
	"context"
	"testing"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Create(ctx context.Context, s *model.Subscription) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}
func (m *mockRepo) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	args := m.Called(ctx, id)
	if sub, ok := args.Get(0).(*model.Subscription); ok {
		return sub, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockRepo) Update(ctx context.Context, s *model.Subscription) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}
func (m *mockRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) List(ctx context.Context, filter repository.ListFilter) ([]*model.Subscription, error) {
	args := m.Called(ctx, filter)
	if subs, ok := args.Get(0).([]*model.Subscription); ok {
		return subs, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockRepo) TotalCostForPeriod(ctx context.Context, from, to time.Time, userID, serviceName *string) (int64, error) {
	args := m.Called(ctx, from, to, userID, serviceName)
	return args.Get(0).(int64), args.Error(1)
}

func TestCreateSubscription_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)
	ctx := context.Background()

	userID := uuid.New().String()
	start := time.Now().UTC()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

	in := service.CreateInput{
		ServiceName: "Netflix",
		Price:       499,
		UserID:      userID,
		StartDate:   start,
		EndDate:     nil,
	}

	sub, err := svc.CreateSubscription(ctx, in)
	assert.NoError(t, err)
	assert.Equal(t, "Netflix", sub.ServiceName)
	assert.Equal(t, userID, sub.UserID)
	assert.WithinDuration(t, start.AddDate(0, 0, 30), *sub.EndDate, time.Second)

	repo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*model.Subscription"))
}

func TestCreateSubscription_InvalidUserID(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)

	in := service.CreateInput{
		ServiceName: "Spotify",
		Price:       100,
		UserID:      "invalid-uuid",
		StartDate:   time.Now(),
	}
	_, err := svc.CreateSubscription(context.Background(), in)
	assert.ErrorIs(t, err, service.ErrInvalid)
}

func TestCreateSubscription_EndBeforeStart(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)

	start := time.Now()
	end := start.AddDate(0, 0, -1)

	in := service.CreateInput{
		ServiceName: "Prime",
		Price:       200,
		UserID:      uuid.New().String(),
		StartDate:   start,
		EndDate:     &end,
	}

	_, err := svc.CreateSubscription(context.Background(), in)
	assert.ErrorIs(t, err, service.ErrInvalid)
}

func TestUpdateSubscription_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)

	existing := &model.Subscription{
		ID:          uuid.New().String(),
		ServiceName: "Netflix",
		Price:       499,
		UserID:      uuid.New().String(),
		StartDate:   time.Now(),
	}

	repo.On("GetByID", mock.Anything, existing.ID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

	newEnd := existing.StartDate.AddDate(0, 0, 15)
	in := service.UpdateInput{
		ServiceName: "Netflix Premium",
		Price:       799,
		UserID:      existing.UserID,
		StartDate:   existing.StartDate,
		EndDate:     &newEnd,
	}

	out, err := svc.UpdateSubscription(context.Background(), existing.ID, in)
	assert.NoError(t, err)
	assert.Equal(t, "Netflix Premium", out.ServiceName)
	assert.Equal(t, 799, out.Price)
	repo.AssertCalled(t, "Update", mock.Anything, mock.AnythingOfType("*model.Subscription"))
}

func TestSumForPeriod_InvalidDates(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)

	from := time.Now()
	to := from.AddDate(0, 0, -1)

	_, err := svc.SumForPeriod(context.Background(), from, to, nil, nil)
	assert.ErrorIs(t, err, service.ErrInvalid)
}

func TestSumForPeriod_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := service.NewSubscriptionService(repo)

	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	repo.On("TotalCostForPeriod", mock.Anything, from, to, (*string)(nil), (*string)(nil)).Return(int64(999), nil)

	total, err := svc.SumForPeriod(context.Background(), from, to, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(999), total)
}
