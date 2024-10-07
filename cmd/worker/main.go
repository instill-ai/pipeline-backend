package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Note: We have temporarily moved the worker process into the API server
	// process due to stability and performance issues. However, we will move it
	// back for a better system architecture. This dummy process will serve as a
	// placeholder until the worker is relocated.

	quitSig := make(chan os.Signal, 1)
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)
	<-quitSig
}
