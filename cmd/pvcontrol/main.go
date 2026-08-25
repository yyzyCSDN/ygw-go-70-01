package main

import (
	"flag"
	"log"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address")
	flag.Parse()
	rt := BuildRuntime()
	server := NewServer(rt)
	log.Printf("pvcontrol listening on %s", *addr)
	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatalf("pvcontrol: %v", err)
	}
}
