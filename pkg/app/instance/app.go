package instance

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grabreflow/pkg/server"
)

type Instance struct {
	ctx    context.Context
	cancel context.CancelFunc
	server *server.Server
}

func NewInstance(bind string, port int) *Instance {
	i := new(Instance)
	i.ctx, i.cancel = context.WithCancel(context.Background())
	i.server = server.NewServer(i, bind, port)

	return i
}

func (i *Instance) Context() context.Context {
	return i.ctx
}

func (i *Instance) Init() error {
	if err := i.server.Init(); err != nil {
		return fmt.Errorf("server: %v", err)
	}
	return nil
}

func (i *Instance) Run() error {
	go i.server.Run()

	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Stop server first so there is not any further request.
	i.server.Stop()
	// Stop others.
	i.cancel()

	// Wait for a while so others can do something.
	time.Sleep(1000 * time.Millisecond)

	return nil
}
