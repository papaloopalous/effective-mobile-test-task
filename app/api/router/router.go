package router

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test_task/app/api/handlers"
	"test_task/app/internal/repo"
	"time"

	"github.com/gorilla/mux"
)

func gracefulStop(repo repo.SubRepo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	repo.Close()
}

func CreateNewRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	subRepo := repo.NewSubRepo("postgres://test:test@localhost:8000/test?sslmode=disable", 200*time.Millisecond) // read_config.GetDBInfo()

	go gracefulStop(subRepo)

	subHandler := &handlers.SubHandler{
		Subs: subRepo,
	}

	router.HandleFunc("/addSub", subHandler.AddSub).Methods("POST")
	router.HandleFunc("/getSub", subHandler.GetByID).Methods("GET")

	return router
}
