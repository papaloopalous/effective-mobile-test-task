package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test_task/app/api/router"
	"test_task/app/internal/logger"
	"time"

	"go.uber.org/zap"
)

func main() {
	srvPort := ":8080" // read_config.GetSrvInfo()

	router := router.CreateNewRouter()

	srv := &http.Server{
		Addr:    srvPort,
		Handler: router,
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
	logger.Log.Info("server is shuttiong down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		logger.Log.Fatal("timeout expired", zap.Error(err))
	}

	logger.Log.Info("server stopped gracefully")
}
