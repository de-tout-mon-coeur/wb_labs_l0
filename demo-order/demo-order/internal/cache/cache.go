package cache

import (
    "sync"
    "encoding/json"
)

type Cache struct {
    mu sync.RWMutex
    m map[string]json.RawMessage
}

func New() *Cache {
    return &Cache{m: make(map[string]json.RawMessage)}
}

func (c *Cache) Set(uid string, raw json.RawMessage) {
    c.mu.Lock(); defer c.mu.Unlock()
    c.m[uid] = raw
}

func (c *Cache) Get(uid string) (json.RawMessage, bool) {
    c.mu.RLock(); defer c.mu.RUnlock()
    v, ok := c.m[uid]
    return v, ok
}

func (c *Cache) LoadFromMap(data map[string]json.RawMessage) {
    c.mu.Lock()
    defer c.mu.Unlock()
    for k,v := range data {
        c.m[k] = v
    }
}
