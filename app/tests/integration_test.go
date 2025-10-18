package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type integAPIResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// doJSON - выполняет HTTP-запрос с JSON и декодирует ответ
func doJSON(method, path string, body any) (*integAPIResp, int, error) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	req, _ := http.NewRequest(method, baseURL()+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c := &http.Client{Timeout: 15 * time.Second}

	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var ar integAPIResp
	_ = json.NewDecoder(resp.Body).Decode(&ar)

	return &ar, resp.StatusCode, nil
}

// doJSONElapsed - то же, что doJSON, но возвращает длительность запроса
func doJSONElapsed(method, path string, body any) (time.Duration, int, error) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	req, _ := http.NewRequest(method, baseURL()+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c := &http.Client{Timeout: 15 * time.Second}

	start := time.Now()
	resp, err := c.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return elapsed, 0, err
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return elapsed, resp.StatusCode, nil
}

// waitServer - ожидает готовность сервера отвечать
func waitServer(t *testing.T) {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		_, code, err := doJSON(http.MethodGet, "/listSubs", map[string]any{"unknown": 1})
		if err == nil && (code == 200 || code == 400 || code == 500) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("server not ready at %s", baseURL())
}

// waitLogFile - ищет лог-файл среди стандартных путей/переменных окружения
func waitLogFile(t *testing.T) string {
	candidates := []string{}

	if s := os.Getenv("LOG_PATH"); s != "" {
		candidates = append(candidates, filepath.Clean(s))
	}

	candidates = append(candidates,
		filepath.Clean("../logs/logs.json"),
		filepath.Clean("./logs/logs.json"),
		filepath.Clean("./../../build/logs/logs.json"),
	)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, p := range candidates {
			if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
				return p
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("log file not found")

	return ""
}

// logContainsSlow - проверяет наличие SLOW-записей в логе
func logContainsSlow(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	s := string(b)
	return strings.Contains(s, "(SLOW)")
}

// TestIntegration_Add_Errors - негативные сценарии для добавления подписок
func TestIntegration_Add_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodPost, "/addSub", map[string]any{"unknown": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": "s", "user_id": uuid.New().String(), "monthly_fee": 10, "start_date": "13-2024", "num_months": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": "s", "user_id": uuid.New().String(), "monthly_fee": 10, "start_date": "01-2024", "num_months": 0})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": "s", "user_id": uuid.New().String(), "monthly_fee": -1, "start_date": "01-2024", "num_months": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}

	_, code, _ = doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": string(long), "user_id": uuid.New().String(), "monthly_fee": 10, "start_date": "01-2024", "num_months": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_Get_Errors - негативные сценарии для получения подписки
func TestIntegration_Get_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodGet, "/getSub?sub_id=bad", nil)
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodGet, "/getSub?sub_id="+uuid.New().String(), nil)
	if code == 200 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_Update_Errors - негативные сценарии обновления
func TestIntegration_Update_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodPut, "/updateSub", map[string]any{"unknown": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPut, "/updateSub", map[string]any{"sub_id": uuid.New().String(), "monthly_fee": 10, "end_date": "13-2024"})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPut, "/updateSub", map[string]any{"sub_id": uuid.New().String(), "monthly_fee": 10, "end_date": "02-2024"})
	if code != 404 {
		t.Fatalf("%d", code)
	}

	userID := uuid.New().String()
	svc := "svc-" + uuid.New().String()
	addBody := map[string]any{"service_name": svc, "user_id": userID, "monthly_fee": 10, "start_date": "01-2024", "num_months": 2}
	addResp, code, err := doJSON(http.MethodPost, "/addSub", addBody)
	if err != nil || code != 201 {
		t.Fatalf("add %d %v", code, err)
	}

	var subID string
	_ = json.Unmarshal(addResp.Data, &subID)
	_, code, _ = doJSON(http.MethodPut, "/updateSub", map[string]any{"sub_id": subID, "monthly_fee": -1, "end_date": "02-2024"})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodPut, "/updateSub", map[string]any{"sub_id": subID, "monthly_fee": 0, "end_date": "12-2023"})
	if code != 400 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_Delete_Errors - негативные сценарии удаления
func TestIntegration_Delete_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodDelete, "/deleteSub?sub_id=bad", nil)
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodDelete, "/deleteSub?sub_id="+uuid.New().String(), nil)
	if code != 200 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_List_Errors - негативные сценарии листинга
func TestIntegration_List_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodGet, "/listSubs", map[string]any{"unknown": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}

	_, code, _ = doJSON(http.MethodGet, "/listSubs", map[string]any{"service_name": string(long), "user_id": "", "start_date": "01-2024", "end_date": "02-2024"})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodGet, "/listSubs", map[string]any{"service_name": "a", "user_id": "bad", "start_date": "01-2024", "end_date": "02-2024"})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodGet, "/listSubs", map[string]any{"service_name": "a", "user_id": "", "start_date": "01-2024", "end_date": "02-2024", "cursor": map[string]any{"last_start": "bad"}})
	if code != 400 {
		t.Fatalf("%d", code)
	}

	_, code, _ = doJSON(http.MethodGet, "/listSubs", map[string]any{"service_name": "a", "user_id": uuid.New().String(), "start_date": "01-2024", "end_date": "02-2024", "cursor": map[string]any{"last_id": "bad"}})
	if code != 400 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_List_Pagination - сценарий с постраничной навигацией
func TestIntegration_List_Pagination(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	userID := uuid.New().String()
	svc := "svc-" + uuid.New().String()
	_, code, err := doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": svc, "user_id": userID, "monthly_fee": 10, "start_date": "01-2024", "num_months": 1})
	if err != nil || code != 201 {
		t.Fatalf("add1 %d %v", code, err)
	}

	_, code, err = doJSON(http.MethodPost, "/addSub", map[string]any{"service_name": svc, "user_id": userID, "monthly_fee": 10, "start_date": "02-2024", "num_months": 1})
	if err != nil || code != 201 {
		t.Fatalf("add2 %d %v", code, err)
	}

	listBody := map[string]any{"service_name": svc, "user_id": userID, "start_date": "01-2024", "end_date": "12-2024", "cursor": map[string]any{"limit": 1}}
	listResp, code, err := doJSON(http.MethodGet, "/listSubs", listBody)
	if err != nil || code != 200 {
		t.Fatalf("list1 %d %v", code, err)
	}

	type curOut struct {
		LastStart string `json:"last_start"`
		LastID    string `json:"last_id"`
		Limit     int    `json:"limit"`
	}

	type listOut struct {
		Cursor *curOut `json:"cursor"`
	}

	var lr listOut
	_ = json.Unmarshal(listResp.Data, &lr)
	if lr.Cursor == nil || lr.Cursor.LastID == "" || lr.Cursor.LastStart == "" {
		t.Fatalf("cursor missing")
	}

	listBody["cursor"] = map[string]any{"last_start": lr.Cursor.LastStart, "last_id": lr.Cursor.LastID, "limit": lr.Cursor.Limit}
	_, code, err = doJSON(http.MethodGet, "/listSubs", listBody)
	if err != nil || code != 200 {
		t.Fatalf("list2 %d %v", code, err)
	}
}

// TestIntegration_Sum_Errors - негативный сценарий суммирования
func TestIntegration_Sum_Errors(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	_, code, _ := doJSON(http.MethodGet, "/totalSubs", map[string]any{"unknown": 1})
	if code != 400 {
		t.Fatalf("%d", code)
	}
}

// TestIntegration_FullFlow - полный сценарий: add-get-update-list-sum-delete
func TestIntegration_FullFlow(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	userID := uuid.New().String()
	svc := "svc-" + uuid.New().String()
	addBody := map[string]any{"service_name": svc, "user_id": userID, "monthly_fee": 10, "start_date": "01-2024", "num_months": 2}
	addResp, code, err := doJSON(http.MethodPost, "/addSub", addBody)
	if err != nil || code != 201 {
		t.Fatalf("add %d %v", code, err)
	}

	var subID string
	_ = json.Unmarshal(addResp.Data, &subID)
	if subID == "" {
		t.Fatalf("empty id")
	}

	_, code, err = doJSON(http.MethodGet, "/getSub?sub_id="+subID, nil)
	if err != nil || code != 200 {
		t.Fatalf("get %d %v", code, err)
	}

	_, code, err = doJSON(http.MethodPut, "/updateSub", map[string]any{"sub_id": subID, "monthly_fee": 20, "end_date": "03-2024"})
	if err != nil || code != 200 {
		t.Fatalf("update %d %v", code, err)
	}

	_, code, err = doJSON(http.MethodGet, "/listSubs", map[string]any{"service_name": svc, "user_id": userID, "start_date": "01-2024", "end_date": "12-2024", "cursor": map[string]any{"limit": 1}})
	if err != nil || code != 200 {
		t.Fatalf("list %d %v", code, err)
	}

	_, code, err = doJSON(http.MethodGet, "/totalSubs", map[string]any{"service_name": svc, "user_id": userID, "start_date": "01-2024", "end_date": "12-2024"})
	if err != nil || code != 200 {
		t.Fatalf("sum %d %v", code, err)
	}

	_, code, err = doJSON(http.MethodDelete, "/deleteSub?sub_id="+subID, nil)
	if err != nil || code != 200 {
		t.Fatalf("delete %d %v", code, err)
	}
}

// TestIntegration_Logs_SlowQueries - проверка отсутствия SLOW-записей
func TestIntegration_Logs_SlowQueries(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" && os.Getenv("INTEGRATION") != "1" {
		t.Skip("")
	}

	waitServer(t)

	logPath := waitLogFile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var maxDur time.Duration
	var mu sync.Mutex
	upd := func(d time.Duration) {
		mu.Lock()
		if d > maxDur {
			maxDur = d
		}
		mu.Unlock()
	}

	errs := make(chan struct{}, 1)
	var wg sync.WaitGroup
	workers := 5
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				d, code, _ := doJSONElapsed(http.MethodGet, "/listSubs", map[string]any{"service_name": "", "user_id": "", "start_date": "01-2024", "end_date": "12-2024", "cursor": map[string]any{"limit": 50}})
				if code == 200 {
					upd(d)
				}
				d, code, _ = doJSONElapsed(http.MethodGet, "/totalSubs", map[string]any{"service_name": "", "user_id": "", "start_date": "01-2024", "end_date": "12-2024"})
				if code == 200 {
					upd(d)
				}
				addBody := map[string]any{"service_name": "svc", "user_id": uuid.New().String(), "monthly_fee": 10, "start_date": "01-2024", "num_months": 2}
				d, code, _ = doJSONElapsed(http.MethodPost, "/addSub", addBody)
				if code == 201 {
					upd(d)
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if logContainsSlow(logPath) {
					errs <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-errs:
		cancel()
		wg.Wait()
		mu.Lock()
		d := maxDur
		mu.Unlock()
		t.Fatalf("slow queries found in logs, max_dur=%s", d)
	case <-ctx.Done():
		cancel()
		wg.Wait()
		mu.Lock()
		d := maxDur
		mu.Unlock()
		t.Logf("max request duration: %s", d)
	}
}
