package main

import (
	"embed"
	"food-lab-chain-service/api"
	"food-lab-chain-service/config"
	"food-lab-chain-service/store"
	"io/fs"
	"log"
	"net/http"
	"strconv"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	port := config.Port()
	log.Printf("food lab chain service listening on :%d", port)
	mux := http.NewServeMux()
	mux.Handle("/api/ops/", http.StripPrefix("/api/ops", opsRouter(newOpsService(nil))))
	mux.Handle("/", api.NewRouter(store.New(), webFS))
	log.Fatal(serveAddress(":"+strconv.Itoa(port), mux))
}
