package mock

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

type logDriver struct {
	*driver
}

func (d logDriver) Dump(ctx context.Context, since, until time.Time, w io.Writer) error {
	return json.NewEncoder(w).Encode(d.driver.log)
}
