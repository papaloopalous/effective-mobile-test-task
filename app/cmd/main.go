package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test_task/app/api/router"
	"time"
)

func main() {
	srvPort := ":8080" // read_config.GetSrvInfo()

	router := router.CreateNewRouter()

	srv := &http.Server{
		Addr:    srvPort,
		Handler: router,
	}

	go func() {
		log.Println("server is starting on port", srvPort)
		err := srv.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("server is shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal("timeout expired: ", err)
	}

	log.Println("server stopped gracefully")
}
