package mock

import (
	"context"
)

type healthDriver struct {
}

func newHealthDriver() *healthDriver {
	return &healthDriver{}
}

func (d *healthDriver) GetHealth(ctx context.Context) error {
	return nil
}
