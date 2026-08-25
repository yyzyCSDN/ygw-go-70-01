package dedup

import "testing"

func TestMarkCheckClear(t *testing.T) {
	cache := NewCache()
	key := cache.Key("inv1", "limit", "payload-a")
	if cache.Check(key) {
		t.Fatal("fresh key must not be marked")
	}
	cache.Mark(key, "inv1|limit")
	if !cache.Check(key) {
		t.Fatal("marked key must be visible")
	}
	cache.ClearGroup("inv1|limit")
	if cache.Check(key) {
		t.Fatal("group-cleared key must not be visible")
	}
}

func TestClearGroup(t *testing.T) {
	cache := NewCache()
	keyA := cache.Key("inv1", "limit", "a")
	keyB := cache.Key("inv1", "limit", "b")
	keyC := cache.Key("inv2", "limit", "c")
	cache.Mark(keyA, "inv1|limit")
	cache.Mark(keyB, "inv1|limit")
	cache.Mark(keyC, "inv2|limit")
	cache.ClearGroup("inv1|limit")
	if cache.Check(keyA) || cache.Check(keyB) {
		t.Fatal("group clear must remove both inv1 keys")
	}
	if !cache.Check(keyC) {
		t.Fatal("other inverter keys must survive group clear")
	}
}
