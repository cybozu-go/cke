package server

import (
	"testing"

	"github.com/cybozu-go/cke"
)

func TestEqualExtenderConfigs(t *testing.T) {
	testCases := []struct {
		message  string
		expected bool
		configs1 []*cke.ExtenderConfig
		configs2 []*cke.ExtenderConfig
	}{
		{
			"valid",
			true,
			[]*cke.ExtenderConfig{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
			[]*cke.ExtenderConfig{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
		},
		{
			"order is difference",
			true,
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
			},
		},
		{
			"empty",
			true,
			[]*cke.ExtenderConfig{},
			[]*cke.ExtenderConfig{},
		},
		{
			"subset",
			false,
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
			},
		},
		{
			"configs1 is empty",
			false,
			[]*cke.ExtenderConfig{},
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
			},
		},
		{
			"difference",
			false,
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:9000",
					FilterVerb: "get",
				},
			},
			[]*cke.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
			},
		},
	}

	for _, tc := range testCases {
		if equalExtenderConfigs(tc.configs1, tc.configs2) != tc.expected {
			t.Error(tc.message)
		}
	}
}
