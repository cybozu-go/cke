package mock

import (
	"context"

	"github.com/cybozu-go/sabakan/v2"
)

func (d *driver) Version(ctx context.Context) (string, error) {
	return sabakan.SchemaVersion, nil
}

func (d *driver) Upgrade(ctx context.Context) error {
	return nil
}
