package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"grabreflow/pkg/app"
	"grabreflow/pkg/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Server struct {
	app     app.App
	addr    string
	router  *gin.Engine
	server  *http.Server
	service *service.Service
}

func NewServer(a app.App, bind string, port int) *Server {
	s := new(Server)
	s.app = a
	s.addr = fmt.Sprintf("%s:%d", bind, port)
	s.router = gin.New()
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}
	s.service = service.NewService()

	s.router.Use(gin.Recovery(), cors.Default())
	s.router.LoadHTMLGlob("view/*")
	s.router.GET("/api/convergence/grabreflow/:sn", s.service.GrabReflow)

	return s
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		logrus.Fatalf("server: stop: %v", err)
	}
	logrus.Debug("server: stopped")
}

func (s *Server) Init() error {
	if err := s.service.Init(s.app.Context()); err != nil {
		return fmt.Errorf("service: %v", err)
	}

	return nil
}

func (s *Server) Run() error {
	logrus.Infof("Server is linstening to %s", s.addr)
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: %v", err)
	}

	return nil
}
