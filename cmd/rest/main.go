package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	httpServer := &http.Server{
		Addr:    defaultPort,
		Handler: router,
	}

	go func() {
		slog.Info("Starting robots server on " + defaultPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := httpServer.Shutdown(context.Background()); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Server exited")
}
