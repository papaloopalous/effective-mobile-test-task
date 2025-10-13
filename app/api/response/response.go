package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// APIResponse определяет структуру ответа API
type APIResponse struct {
	Code    int         `json:"code"`           // HTTP код ответа
	Message string      `json:"message"`        // Сообщение для пользователя
	Data    interface{} `json:"data,omitempty"` // Данные ответа (опционально)
}

// WriteAPIResponse формирует и отправляет JSON-ответ клиенту
// Устанавливает заголовки ответа, сериализует данные и логирует ошибки при неудаче
func WriteAPIResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := APIResponse{
		Code:    statusCode,
		Message: message,
		Data:    data,
	}

	log.Println("response sent: ", resp)

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Println("failed to write a response: ", err)
	}
}
