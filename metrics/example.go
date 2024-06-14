package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var port = flag.String("port", "2112", "Port to listen on")

func main() {
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())

	log.Printf("listening on :%s\n", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
