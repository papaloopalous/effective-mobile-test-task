package readconfig

import (
	"net/url"
	"time"

	"task_test/internal/logger"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Инициализация конфигурации из файла ./configs/config.yaml
func init() {
	viper.SetConfigFile("./configs/config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		logger.Log.Fatal("failed to load config", zap.Error(err))
	}
}

// GetSrvInfo - получить порт сервера и таймаут чтения заголовка
func GetSrvInfo() (string, time.Duration) {
	port := viper.GetString("server_port")
	timeout := viper.GetDuration("read_header_timeout")
	return port, timeout
}

// GetDBInfo - сформировать DSN для PostgreSQL и вернуть порог медленного запроса
func GetDBInfo() (string, time.Duration, time.Duration) {
	dbUser := viper.GetString("db.user")
	dbPass := viper.GetString("db.password")
	dbHost := viper.GetString("db.host")
	dbPort := viper.GetString("db.port")
	dbName := viper.GetString("db.name")
	dbTimeout := viper.GetDuration("db.timeout")
	sslMode := viper.GetString("db.sslmode")
	minConns := viper.GetString("db.pool.min_conns")
	maxConns := viper.GetString("db.pool.max_conns")
	maxConnLifetime := viper.GetString("db.pool.max_conn_lifetime")
	maxConnIdleTime := viper.GetString("db.pool.max_conn_idle_time")
	healthCheckPeriod := viper.GetString("db.pool.health_check_period")
	slowThreshold := viper.GetDuration("slow_threshold")

	u := url.URL{
		Scheme: "postgres",
		Path:   "/" + dbName,
		Host:   dbHost + ":" + dbPort,
		User:   url.UserPassword(dbUser, dbPass),
	}

	q := url.Values{}
	q.Set("sslmode", sslMode)
	q.Set("pool_min_conns", minConns)
	q.Set("pool_max_conns", maxConns)
	q.Set("pool_max_conn_lifetime", maxConnLifetime)
	q.Set("pool_max_conn_idle_time", maxConnIdleTime)
	q.Set("pool_health_check_period", healthCheckPeriod)

	u.RawQuery = q.Encode()

	return u.String(), slowThreshold, dbTimeout
}
