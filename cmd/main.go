package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	// Read PORT from environment (required by Cloud Run)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default for local testing
	}

	log.Printf("Server starting on port %s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
