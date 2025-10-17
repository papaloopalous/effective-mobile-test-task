package repo

import (
	"time"

	"github.com/google/uuid"
)

type SubRepo interface {
	Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error)
	Read(subID uuid.UUID, format string) (SubInfo, error)
	Update(subID uuid.UUID, monthlyFee int, endDate time.Time) error
	Delete(subID uuid.UUID) error
	List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string, cur PageCursor) ([]SubInfo, *PageCursor, error)
	GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error)
	Close()
}
