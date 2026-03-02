package cache

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New(1 * time.Second)
	defer c.Stop()

	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("got %v, want value1", val)
	}
}

func TestCacheMiss(t *testing.T) {
	c := New(1 * time.Second)
	defer c.Stop()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New(50 * time.Millisecond)
	defer c.Stop()

	c.Set("key1", "value1")

	time.Sleep(100 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := New(10 * time.Second)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Invalidate("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be invalidated")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	c := New(10 * time.Second)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.InvalidateAll()

	_, ok1 := c.Get("key1")
	_, ok2 := c.Get("key2")
	if ok1 || ok2 {
		t.Error("expected all keys to be invalidated")
	}
}
