package router

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"syscall"
	"time"

	"task_test/api/handlers"
	_ "task_test/docs"
	"task_test/internal/logger"
	readConfig "task_test/internal/read_config"
	"task_test/internal/repo"
	"task_test/util"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

func stopRouter(ctx context.Context, wg *sync.WaitGroup, repo repo.SubRepo) {
	<-ctx.Done()
	logger.Log.Info("closing repo and logger")
	repo.Close()
	err := logger.Log.Sync()
	if err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTTY) {
		logger.Log.Error(util.ErrLogLoggerSync, zap.Error(err))
	}
	defer wg.Done()
}

func CreateNewRouter(ctx context.Context, wg *sync.WaitGroup) *mux.Router {
	router := mux.NewRouter()

	repoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	dsn, slowThreshold, timeout := readConfig.GetDBInfo()
	subRepo := repo.NewSubRepo(repoCtx, dsn, slowThreshold)

	go stopRouter(ctx, wg, subRepo)

	subHandler := &handlers.SubHandler{
		Subs:    subRepo,
		Timeout: timeout,
	}

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Methods("GET")

	router.HandleFunc("/addSub", subHandler.AddSub).Methods("POST")
	router.HandleFunc("/getSub", subHandler.GetByID).Methods("GET")
	router.HandleFunc("/updateSub", subHandler.UpdateByID).Methods("PUT")
	router.HandleFunc("/deleteSub", subHandler.RemoveByID).Methods("DELETE")
	router.HandleFunc("/listSubs", subHandler.ListSubs).Methods("GET", "POST")
	router.HandleFunc("/totalSubs", subHandler.SumSubs).Methods("GET", "POST")

	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return router
}
