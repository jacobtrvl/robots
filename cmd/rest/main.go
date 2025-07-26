package main

import (
	"log/slog"
	"net/http"

	"github.com/jacobtrvl/robots/pkg/api"
	"github.com/jacobtrvl/robots/pkg/robot"
)

const (
	defaultPort = ":8080"
)

func main() {
	w := robot.NewMockWarehouse()
	server := api.NewRobotApi(w)
	router := server.NewRouter()
	http.Handle("/", router)
	if err := http.ListenAndServe(defaultPort, nil); err != nil {
		slog.Error("failed to start server", "error", err)
		return
	}
  
}
