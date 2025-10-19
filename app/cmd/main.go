// Сервис подписок (Subscriptions Service)
//
// @title        API Подписок
// @version      1.0
// @description  API для управления пользовательскими подписками: создание, получение, обновление, удаление, список с пагинацией и подсчёт суммы по периодам.
// @BasePath     /
// @schemes      http
//
// @tag.name Subscriptions
// @tag.description Операции с подписками пользователей: создание, получение, обновление, удаление, список с пагинацией и подсчёт суммарной стоимости.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"task_test/api/router"
	"task_test/internal/logger"
	readConfig "task_test/internal/read_config"

	"go.uber.org/zap"
)

func main() {
	srvPort, timeout := readConfig.GetSrvInfo()

	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	router := router.CreateNewRouter(ctx, wg)

	srv := &http.Server{
		ReadHeaderTimeout: timeout,
		Addr:              srvPort,
		Handler:           router,
	}

	go func() {
		logger.Log.Info("server is starting", zap.String("port", srvPort))
		err := srv.ListenAndServe()

		if err != nil && errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	cancel()

	wg.Wait()
	logger.Log.Info("server is shutting down")
	ctxMain, cancelMain := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMain()

	if err := srv.Shutdown(ctxMain); err != nil {
		logger.Log.Fatal("timeout expired", zap.Error(err))
	}

	logger.Log.Info("server stopped gracefully")
}
