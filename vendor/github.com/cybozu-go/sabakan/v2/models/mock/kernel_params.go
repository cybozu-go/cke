package mock

import (
	"context"
	"sync"

	"github.com/cybozu-go/sabakan/v2"
)

type kernelParamsDriver struct {
	mu           sync.Mutex
	kernelParams map[string]string
}

func newKernelParamsDriver() *kernelParamsDriver {
	return &kernelParamsDriver{
		kernelParams: make(map[string]string),
	}
}

func (d *kernelParamsDriver) PutParams(ctx context.Context, os string, params string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.kernelParams[os] = params
	return nil
}

func (d *kernelParamsDriver) GetParams(ctx context.Context, os string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if val, ok := d.kernelParams[os]; ok {
		return val, nil
	}

	return "", sabakan.ErrNotFound
}
