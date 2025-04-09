package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"websocket_try3/client"
	"websocket_try3/hub"
)

func main() {
	hubs := hub.NewHub()
	go hubs.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		client.ServeWs(w, r, hubs)
	})

	srv := &http.Server{
		Addr: ":8080",
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()
	<-done

	log.Println("Shutting down server...")
}
