package repo

import "github.com/google/uuid"

type SubRepo interface {
	Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate string, nMonths int) (uuid.UUID, error)
	Read(subID uuid.UUID) (SubInfo, error)
	Update(subID uuid.UUID, monthlyFee int, endDate string) error
	Delete(subID uuid.UUID) error
	List(serviceName string, userID uuid.UUID, startDate string, endDate string) []SubInfo
	GetSum(serviceName string, userID uuid.UUID, startDate string, endDate string) int64
	Close()
}
