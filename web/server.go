package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// Server type handles the core HTTP requests for the `pictly` app
type Server struct {
	s *http.Server
}

// NewServer constructs a web server which can then be invoked via the `Server.Run()` command.
func NewServer(router http.Handler, port int) (*Server, error) {
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return &Server{
		s: s,
	}, nil
}

// Run will start the server and wait for error or shutdown - this will block the caller. Any error encountered
// during server startup or operation will be returned to the caller here. The server can be exit'ed using a
// SIGINT, SIGTERM or SIGQUIT interrupt
func (s *Server) Run() error {
	term := make(chan os.Signal, 1)
	errch := make(chan error, 1)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		log.Infof("starting server: host = %s", s.s.Addr)

		if err := s.s.ListenAndServe(); err != nil {
			log.Errorf("failed to run server: error = %s", err)
			errch <- err
		}
	}()

	var err error
	select {
	case tsig := <-term:
		log.Infof("closing server: signal = %s", tsig)
	case err = <-errch:
		log.Errorf("error starring server : error = %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.s.Shutdown(ctx)
	return err
}
