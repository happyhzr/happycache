package happycache

import (
	"errors"
	"log"
	"sync"

	pb "github.com/happyhzr/happycache/happycachepb"
	"github.com/happyhzr/happycache/singleflight"
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
	peers     PeerPicker
	loader    *singleflight.Group
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
		loader:    &singleflight.Group{},
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
	view, err := g.loader.Do(key, func() (any, error) {
		if g.peers != nil {
			peer, ok := g.peers.PickPeer(key)
			if ok {
				value, err := g.getFromPeer(peer, key)
				if err == nil {
					return value, nil
				}
				log.Println("[happycache] failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return view.(*ByteView), nil
	}
	return nil, err
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (*ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return nil, err
	}
	return &ByteView{b: res.Value}, nil
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

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("registerPeerPick called more than once")
	}
	g.peers = peers
}
