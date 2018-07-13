package cli

import (
	"strings"

	"github.com/cybozu-go/cke"
)

var storage cke.Storage

// Setup setups this package.
func Setup(s cke.Storage) {
	storage = s
}

type commaStrings []string

func (o *commaStrings) String() string {
	return strings.Join([]string(*o), ",")
}

func (o *commaStrings) Set(v string) error {
	*o = strings.Split(v, ",")
	return nil
}
