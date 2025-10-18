package repo

import (
	"time"

	"github.com/google/uuid"
)

type SubRepo interface {
	// Create - создать подписку
	Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error)
	// Read - получить информацию о подписке по ID
	Read(subID uuid.UUID, format string) (SubInfo, error)
	// Update - обновить стоимость/дату окончания подписки
	Update(subID uuid.UUID, monthlyFee int, endDate time.Time) error
	// Delete - удалить подписку
	Delete(subID uuid.UUID) error
	// List - получить список подписок с фильтрами и курсором
	List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string, cur PageCursor) ([]SubInfo, *PageCursor, error)
	// GetSum - получить суммарную стоимость по фильтрам
	GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error)
	// Close - закрыть подключение/ресурсы
	Close()
}
