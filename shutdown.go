package goweb

import (
	"sync"
	"os"
	"os/signal"
	"syscall"
	"github.com/sirupsen/logrus"
	"time"
	"context"
)

var (
	hooks    []func()
	mutex    sync.Mutex
	quitChan chan struct{}
)

func init() {
	quitChan = make(chan struct{})
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	logrus.Debug("Init shutdown hooks handler...")

	go func() {
		<-signalChan
		logrus.Info("Executing shutdown hooks...")
		startTime := time.Now()
		for i := len(hooks) - 1; i >= 0; i-- {
			hooks[i]()
		}
		logrus.Infof("Shutdown hooks finished in %v", time.Since(startTime).Truncate(time.Millisecond))
		close(quitChan)
	}()
}

func OnShutdown(hook func()) {
	mutex.Lock()
	hooks = append(hooks, hook)
	mutex.Unlock()
}

func WaitShutdown(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-quitChan:
	}
}
