package shutdown

import (
	"CodeBorrowing/pkg/logger"
	"os"
	"os/signal"
)

func Graceful(appLogger *logger.Logger, signals []os.Signal, quit chan<- interface{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	sig := <-ch
	appLogger.Infof("SHUTDOWN: Caught signal %v", sig)
	quit <- nil
}
