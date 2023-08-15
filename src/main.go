package main

import "log"

func main() {
    storage, err := newPostgresStore()
    if err != nil {
        log.Fatal(err)
    }
    if err := storage.Init(); err != nil {
        log.Fatal(err)
    }
    server := newServer("8080", storage)
    server.Run()
}
