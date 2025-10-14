package router

import (
	"os"
	"os/signal"
	"syscall"
	"test_task/app/api/handlers"
	"test_task/app/internal/logger"
	readConfig "test_task/app/internal/read_config"
	"test_task/app/internal/repo"

	"github.com/gorilla/mux"
)

func gracefulStop(repo repo.SubRepo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("closing repo and logger")
	repo.Close()
	logger.Log.Sync()
}

func CreateNewRouter() *mux.Router {
	router := mux.NewRouter()

	subRepo := repo.NewSubRepo(readConfig.GetDBInfo())

	go gracefulStop(subRepo)

	subHandler := &handlers.SubHandler{
		Subs: subRepo,
	}

	router.HandleFunc("/addSub", subHandler.AddSub).Methods("POST")
	router.HandleFunc("/getSub", subHandler.GetByID).Methods("GET")
	router.HandleFunc("/updateSub", subHandler.UpdateByID).Methods("PUT")
	router.HandleFunc("/deleteSub", subHandler.RemoveByID).Methods("DELETE")
	router.HandleFunc("/listSubs", subHandler.ListSubs).Methods("GET")
	router.HandleFunc("/totalSubs", subHandler.SumSubs).Methods("GET")

	return router
}
