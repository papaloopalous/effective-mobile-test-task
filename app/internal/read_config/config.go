package readconfig

import (
	"fmt"
	"task_test/internal/logger"

	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	viper.SetConfigFile("./configs/config.yaml")

	err := viper.ReadInConfig()
	if err != nil {
		logger.Log.Fatal("failed to load config", zap.Error(err))
	}
}

func GetSrvInfo() (string, time.Duration) {
	port := viper.GetString("server_port")
	timeout := viper.GetDuration("read_header_timeout")
	return port, timeout
}

func GetDBInfo() (string, time.Duration) {
	dsnTemplate := "postgres://%s:%s@%s%s/%s?sslmode=disable"
	dbUser := viper.GetString("db.user")
	dbPass := viper.GetString("db.password")
	dbHost := viper.GetString("db.host")
	dbPort := viper.GetString("db.port")
	dbName := viper.GetString("db.name")

	slowThreshold := viper.GetDuration("slow_threshold")

	return fmt.Sprintf(dsnTemplate, dbUser, dbPass, dbHost, dbPort, dbName), slowThreshold
}
