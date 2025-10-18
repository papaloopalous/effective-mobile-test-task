package router

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

func gracefulStop(repo repo.SubRepo) {
	// останавливаем приложение по сигналам (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("closing repo and logger")
	repo.Close()
	err := logger.Log.Sync()
	if err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTTY) {
		logger.Log.Error(util.ErrLogLoggerSync, zap.Error(err))
	}
}

func CreateNewRouter() *mux.Router {
	router := mux.NewRouter()

	subRepo := repo.NewSubRepo(readConfig.GetDBInfo())

	go gracefulStop(subRepo)

	subHandler := &handlers.SubHandler{
		Subs: subRepo,
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
