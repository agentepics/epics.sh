package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/agentepics/epics.sh/internal/daemon"
	"github.com/agentepics/epics.sh/internal/daemon/store"
)

func main() {
	home, err := store.ResolveHome()
	if err != nil {
		log.Fatal(err)
	}
	server, err := daemon.New(daemon.Options{Home: home})
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
