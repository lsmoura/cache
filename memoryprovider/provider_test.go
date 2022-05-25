package memoryprovider

import (
	"context"
	"testing"
)

func TestMemoryProvider_SetGet(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("MemoryProvider is nil")
	}

	const testKey = "key"
	const testValue = "value"

	if err := provider.Set(context.Background(), testKey, []byte(testValue), 0); err != nil {
		t.Fatal("cannot set value", err)
	}

	if value, err := provider.Get(context.Background(), testKey); err != nil {
		t.Fatal("cannot get value", err)
	} else if string(value) != testValue {
		t.Fatal("value is not equal to the original set value")
	}
}

func TestMemoryProvider_NilProvider(t *testing.T) {
	provider := &MemoryProvider{}

	if err := provider.Set(context.Background(), "key", nil, 0); err == nil {
		t.Fatal("uninitialized provider should return error")
	}

	if _, err := provider.Get(context.Background(), "key"); err == nil {
		t.Fatal("uninitialized provider should return error")
	}
}

func TestMemoryProvider_UnsetKey(t *testing.T) {
	provider := New()

	const testKey = "key"
	value, err := provider.Get(context.Background(), testKey)
	if err != nil {
		t.Fatal("cannot get value", err)
	}
	if value != nil {
		t.Fatal("value should be nil")
	}
}
