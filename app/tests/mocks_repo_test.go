package tests

import (
	repoPkg "task_test/internal/repo"
	"time"

	"github.com/google/uuid"
)

type mockSubRepo struct {
	createFn func(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error)
	readFn   func(subID uuid.UUID, format string) (repoPkg.SubInfo, error)
	updateFn func(subID uuid.UUID, monthlyFee int, endDate time.Time) error
	deleteFn func(subID uuid.UUID) error
	listFn   func(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string, cur repoPkg.PageCursor) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error)
	sumFn    func(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error)
}

func (m *mockSubRepo) Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(serviceName, monthlyFee, userID, startDate, nMonths)
	}

	return uuid.Nil, nil
}

func (m *mockSubRepo) Read(subID uuid.UUID, format string) (repoPkg.SubInfo, error) {
	if m.readFn != nil {
		return m.readFn(subID, format)
	}

	return repoPkg.SubInfo{}, nil
}

func (m *mockSubRepo) Update(subID uuid.UUID, monthlyFee int, endDate time.Time) error {
	if m.updateFn != nil {
		return m.updateFn(subID, monthlyFee, endDate)
	}

	return nil
}

func (m *mockSubRepo) Delete(subID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(subID)
	}

	return nil
}

func (m *mockSubRepo) List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string, cur repoPkg.PageCursor) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error) {
	if m.listFn != nil {
		return m.listFn(serviceName, userID, startDate, endDate, format, cur)
	}

	return nil, nil, nil
}

func (m *mockSubRepo) GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error) {
	if m.sumFn != nil {
		return m.sumFn(serviceName, userID, startDate, endDate)
	}

	return 0, nil
}

func (m *mockSubRepo) Close() {}
