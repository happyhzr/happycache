package happycache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	v, _ := f.Get("key")
	if !reflect.DeepEqual(v, expect) {
		t.Fatal("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	c := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		v, ok := db[key]
		if ok {
			_, ok = loadCounts[key]
			if !ok {
				loadCounts[key] = 0
			}
			loadCounts[key] += 1
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))
	for k, v := range db {
		view, err := c.Get(k)
		if err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		_, err = c.Get(k)
		if err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss ", k)
		}
	}
	view, err := c.Get("unknown")
	if err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", view)
	}
}
