package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    stan "github.com/nats-io/stan.go"
    "demo-order/internal/db"
    "demo-order/internal/cache"
)

var (
    natsURL = "nats://localhost:4222" // при локальной разработке через Docker Compose
    clusterID = "test-cluster"
    clientID = "order-service-1" // можно сделать уникальным
    subject = "orders"           // канал
)

func main() {
    // конфиг через env
    pgdsn := os.Getenv("PG_DSN")
    if pgdsn == "" {
        pgdsn = "postgres://demo_user:demo_pass@localhost:5432/demo_db?sslmode=disable"
    }
    // init db
    store, err := db.New(pgdsn)
    if err != nil {
        log.Fatalf("db.New: %v", err)
    }

    // init cache and restore from DB
    c := cache.New()
    ctx := context.Background()
    rows, err := store.LoadAllOrders(ctx)
    if err != nil {
        log.Printf("warning: failed load orders: %v", err)
    } else {
        c.LoadFromMap(rows)
        log.Printf("cache restored: %d orders", len(rows))
    }

    // connect to nats-streaming
    sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
    if err != nil {
        log.Fatalf("stan.Connect: %v", err)
    }
    defer sc.Close()

    // durable subscription, manual ack
    _, err = sc.Subscribe(subject, func(m *stan.Msg) {
        // process message
        var raw map[string]interface{}
        if err := json.Unmarshal(m.Data, &raw); err != nil {
            log.Printf("invalid json: %v; msg sequence: %d", err, m.Sequence)
            // don't ack => will be redelivered (or consider acking and logging)
            return
        }
        // minimal validation
        uid, ok := raw["order_uid"].(string)
        if !ok || uid == "" {
            log.Printf("missing order_uid: seq=%d", m.Sequence)
            // reject by ack to avoid poisoning? we'll ack to skip bad data
            m.Ack()
            return
        }
        track, _ := raw["track_number"].(string)

        // write to DB (transactional for safety)
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := store.UpsertOrder(ctx, uid, track, raw); err != nil {
            log.Printf("db upsert error for %s: %v", uid, err)
            // don't ack -> redeliver
            return
        }
        // write to cache
        b, _ := json.Marshal(raw)
        c.Set(uid, b)
        // ack message
        if err := m.Ack(); err != nil {
            log.Printf("ack error: %v", err)
        } else {
            log.Printf("processed order %s (seq=%d)", uid, m.Sequence)
        }
    }, stan.DurableName("order-durable"), stan.SetManualAckMode(), stan.AckWait(60*time.Second))
    if err != nil {
        log.Fatalf("Subscribe error: %v", err)
    }

    // HTTP handlers
    http.HandleFunc("/order/", func(w http.ResponseWriter, r *http.Request) {
        // path: /order/{id}
        id := r.URL.Path[len("/order/"):]
        if id == "" {
            http.Error(w, "missing id", http.StatusBadRequest)
            return
        }
        if v, ok := c.Get(id); ok {
            w.Header().Set("Content-Type", "application/json")
            w.Write(v)
            return
        }
        // try load from DB as fallback
        raw, err := store.GetOrder(r.Context(), id)
        if err == nil {
            // update cache
            c.Set(id, raw)
            w.Header().Set("Content-Type", "application/json")
            w.Write(raw)
            return
        }
        http.NotFound(w, r)
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // простая веб-страница (встраиваем небольшую форму)
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprint(w, `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Order viewer</title></head>
<body>
  <h1>Order viewer</h1>
  <form id="f">
    <input id="id" placeholder="order_uid" style="width:400px"/>
    <button type="submit">Get</button>
  </form>
  <pre id="out" style="white-space:pre-wrap;border:1px solid #ddd;padding:10px;margin-top:10px"></pre>
  <script>
    document.getElementById('f').onsubmit = async e => {
      e.preventDefault();
      const id = document.getElementById('id').value.trim();
      if(!id) return;
      const res = await fetch('/order/'+encodeURIComponent(id));
      if(res.status==200){
        const t = await res.text();
        document.getElementById('out').textContent = t;
      } else {
        document.getElementById('out').textContent = 'Not found ('+res.status+')';
      }
    }
  </script>
</body>
</html>`)
    })

    addr := ":8080"
    log.Printf("http server listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}
