package main

import (
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/hoorayman/moss/pkg/moss"
)

func main() {
	log.Print("Moss starting...")
	moss := moss.NewMoss()

	moss.Start()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal
	moss.Stop()
}
