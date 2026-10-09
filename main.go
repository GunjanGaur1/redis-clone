package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"redis-clone/internal/server"
	"redis-clone/internal/store"
)

func main() {
	addr := flag.String("addr", ":6379", "TCP address to listen on")
	dataFile := flag.String("data", "dump.json", "snapshot file path")
	flag.Parse()

	db := store.New()
	if err := db.Load(*dataFile); err != nil {
		log.Printf("snapshot not loaded: %v", err)
	}

	srv := server.New(*addr, db, *dataFile)
	go func() {
		log.Printf("Redis Clone listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	if err := db.Save(*dataFile); err != nil {
		log.Printf("final snapshot failed: %v", err)
	}
	log.Println("Redis Clone stopped")
}
