package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task_test/api/router"
	"task_test/internal/logger"
	readConfig "task_test/internal/read_config"

	"time"

	"go.uber.org/zap"
)

func main() {
	srvPort, timeout := readConfig.GetSrvInfo()

	router := router.CreateNewRouter()

	srv := &http.Server{
		ReadHeaderTimeout: timeout,
		Addr:              srvPort,
		Handler:           router,
	}

	go func() {
		logger.Log.Info("server is starting", zap.String("port", srvPort))
		err := srv.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Log.Info("server is shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		logger.Log.Fatal("timeout expired", zap.Error(err))
	}

	logger.Log.Info("server stopped gracefully")
}
