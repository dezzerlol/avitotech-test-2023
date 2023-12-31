package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dezzerlol/avitotech-test-2023/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
	worker worker.TaskDistributor
}

func New(logger *zap.SugaredLogger, db *pgxpool.Pool, worker worker.TaskDistributor) *Server {
	return &Server{
		logger: logger,
		db:     db,
		worker: worker,
	}
}

func (s Server) Run(host, port string) {
	router := s.setHTTPRouter()

	addr := fmt.Sprintf("%s:%s", host, port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)

		// Listen for SIGNINT and SIGTERM signals
		// and write them in quit channel
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		s.logger.Infoln("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		s.logger.Infoln("Completing background tasks...")

		err := srv.Shutdown(ctx)

		if err != nil {
			shutdownError <- err
		}

		shutdownError <- nil
	}()

	s.logger.Infoln("Starting server on: ", addr)
	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		s.logger.Fatal(err)
	}

	err = <-shutdownError
	if err != nil {
		s.logger.Fatal(err)
	}

	s.logger.Infoln("Stopped server")
}
