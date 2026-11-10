package happycache

import (
	"errors"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache *cache
}

var mu sync.RWMutex
var groups = map[string]*Group{}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: &cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

func (g *Group) Get(key string) (*ByteView, error) {
	if key == "" {
		return nil, errors.New("key is required")
	}
	v, ok := g.mainCache.get(key)
	if ok {
		log.Printf("[happycache] key: %s hit\n", key)
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (*ByteView, error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (*ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return nil, err
	}
	value := &ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value *ByteView) {
	g.mainCache.add(key, value)
}
