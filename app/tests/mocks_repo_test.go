package tests

// Моки репозитория для изоляции хендлеров и тестов репозитория.

import (
	"context"
	"time"

	repoPkg "task_test/internal/repo"

	"github.com/google/uuid"
)

type mockSubRepo struct {
	createFn func(ctx context.Context, args repoPkg.CreateArgs) (uuid.UUID, error)
	readFn   func(ctx context.Context, subID uuid.UUID, format string) (repoPkg.SubInfo, error)
	updateFn func(ctx context.Context, subID uuid.UUID, monthlyFee int, endDate time.Time) error
	deleteFn func(ctx context.Context, subID uuid.UUID) error
	listFn   func(ctx context.Context, args repoPkg.ListArgs) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error)
	sumFn    func(ctx context.Context, serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error)
}

// Create - заглушка создания подписки
func (m *mockSubRepo) Create(ctx context.Context, args repoPkg.CreateArgs) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(ctx, args)
	}

	return uuid.Nil, nil
}

// Read - заглушка чтения подписки
func (m *mockSubRepo) Read(ctx context.Context, subID uuid.UUID, format string) (repoPkg.SubInfo, error) {
	if m.readFn != nil {
		return m.readFn(ctx, subID, format)
	}

	return repoPkg.SubInfo{}, nil
}

// Update - заглушка обновления подписки
func (m *mockSubRepo) Update(ctx context.Context, subID uuid.UUID, monthlyFee int, endDate time.Time) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, subID, monthlyFee, endDate)
	}

	return nil
}

// Delete - заглушка удаления подписки
func (m *mockSubRepo) Delete(ctx context.Context, subID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, subID)
	}

	return nil
}

// List - заглушка листинга подписок
func (m *mockSubRepo) List(ctx context.Context, args repoPkg.ListArgs) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error) {
	if m.listFn != nil {
		return m.listFn(ctx, args)
	}

	return nil, nil, nil
}

// GetSum - заглушка суммирования подписок
func (m *mockSubRepo) GetSum(ctx context.Context, serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error) {
	if m.sumFn != nil {
		return m.sumFn(ctx, serviceName, userID, startDate, endDate)
	}

	return 0, nil
}

// Close - заглушка закрытия
func (m *mockSubRepo) Close() {}
