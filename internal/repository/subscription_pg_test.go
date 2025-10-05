package repository_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newMock() (*sql.DB, sqlmock.Sqlmock, repository.SubscriptionRepo) {
	db, mock, _ := sqlmock.New()
	repo := repository.NewPGRepo(db)
	return db, mock, repo
}

func TestCreate_Success(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	sub := &model.Subscription{
		ID:          uuid.New().String(),
		ServiceName: "Netflix",
		Price:       499,
		UserID:      uuid.New().String(),
		StartDate:   time.Now(),
		EndDate:     nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO subscriptions`)).
		WithArgs(sub.ID, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate, sub.CreatedAt, sub.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_Found(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	id := uuid.New().String()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow(id, "Spotify", int64(299), uuid.New().String(), now, now.AddDate(0, 1, 0), now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(rows)

	sub, err := repo.GetByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, id, sub.ID)
	assert.Equal(t, "Spotify", sub.ServiceName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	id := uuid.New().String()
	mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE id =").
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	sub, err := repo.GetByID(context.Background(), id)
	assert.Nil(t, sub)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_Success(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	sub := &model.Subscription{
		ID:          uuid.New().String(),
		ServiceName: "YouTube",
		Price:       999,
		UserID:      uuid.New().String(),
		StartDate:   time.Now(),
		EndDate:     nil,
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET service_name=$1, price=$2, user_id=$3, start_date=$4, end_date=$5, updated_at=$6 WHERE id=$7`)).
		WithArgs(sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate, sub.UpdatedAt, sub.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_NotFound(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	sub := &model.Subscription{ID: uuid.New().String()}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(context.Background(), sub)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_Success(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	id := uuid.New().String()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM subscriptions WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), id)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_NotFound(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	id := uuid.New().String()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM subscriptions WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), id)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Success(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow(uuid.New().String(), "Netflix", int64(499), uuid.New().String(), now, now, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10, 0).
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), repository.ListFilter{Limit: 10, Offset: 0})
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "Netflix", list[0].ServiceName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTotalCostForPeriod_Success(t *testing.T) {
	db, mock, repo := newMock()
	defer db.Close()

	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()
	var total int64 = 1500

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(SUM(price),0)`)).
		WithArgs(to, from, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(total))

	got, err := repo.TotalCostForPeriod(context.Background(), from, to, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, total, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}
