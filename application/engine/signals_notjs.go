//go:build !js

package engine

import (
	"os"
	"os/signal"
	"syscall"
)

// watchSignals listens for SIGINT/SIGTERM and closes shutdownCh when received.
func watchSignals(shutdownCh chan struct{}) {
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		signal.Stop(sigCh)
		close(shutdownCh)
	}()
}
