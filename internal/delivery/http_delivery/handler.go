package http_delivery

import (
	"log"
	"net/http"
	"os"
	"websocket_try3/internal/config"
	"websocket_try3/internal/delivery/websocket"
	"websocket_try3/internal/repository"
	"websocket_try3/internal/usecase"

	"github.com/redis/go-redis/v9"
)

func Handler() *http.ServeMux {
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "",
		DB:       0, // use default DB
	})

	hub := websocket.NewHub()

	go hub.Run(rdb)

	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	roomRepo := repository.NewRoomRepository(db)

	wsUsecase := usecase.NewWebSocketUsecase(userRepo, messageRepo, roomRepo)

	wsHandler := websocket.NewWebSocketHandler(wsUsecase)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.ServeWS(w, r, hub)
	})

	return mux
}
