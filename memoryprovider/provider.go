package memoryprovider

import (
	"context"
	"fmt"
	"time"
)

type MemoryProvider struct {
	data map[string][]byte
}

func New() *MemoryProvider {
	return &MemoryProvider{
		data: make(map[string][]byte),
	}
}

func (p *MemoryProvider) Get(_ context.Context, key string) ([]byte, error) {
	if p.data == nil {
		return nil, fmt.Errorf("memory provider is not initialized")
	}
	data, ok := p.data[key]
	if !ok {
		return nil, nil
	}

	return data, nil
}

func (p *MemoryProvider) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if p.data == nil {
		return fmt.Errorf("memory provider is not initialized")
	}
	p.data[key] = value
	return nil
}
