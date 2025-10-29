package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "time"

    stan "github.com/nats-io/stan.go"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("usage: publish <file.json>")
    }

    file := os.Args[1]
    data, err := ioutil.ReadFile(file)
    if err != nil {
        log.Fatal(err)
    }

    sc, err := stan.Connect("test-cluster", "publisher-"+randomSuffix(), stan.NatsURL("nats://localhost:4222"))
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    if err := sc.Publish("orders", data); err != nil {
        log.Fatalf("publish error: %v", err)
    }

    log.Println("published successfully 🎉")
}

func randomSuffix() string {
    return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}

