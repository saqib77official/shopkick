package main

import (
    "net/http"
    "log"
)

func main() {
    // Serve static files from /static folder
    http.Handle("/", http.FileServer(http.Dir("./static")))

    // Future API example (add more as needed)
    // http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
    //     w.Write([]byte("OK"))
    // })

    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}