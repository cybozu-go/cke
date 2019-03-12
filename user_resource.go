package cke

import (
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RegisterResource(data []byte) error {
	var resource struct {
		v1.TypeMeta
		v1.ObjectMeta
	}

	err := yaml.Unmarshal(data, &resource)
	if err != nil {
		return err
	}

	return nil
}
