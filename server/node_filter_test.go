package server

import (
	"testing"

	"github.com/cybozu-go/cke/scheduler"
)

func TestEqualExtenderConfigs(t *testing.T) {
	testCases := []struct {
		message  string
		expected bool
		configs1 []*scheduler.ExtenderConfig
		configs2 []*scheduler.ExtenderConfig
	}{
		{
			"valid",
			true,
			[]*scheduler.ExtenderConfig{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
			[]*scheduler.ExtenderConfig{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
		},
		{
			"if order is difference, it should return false",
			false,
			[]*scheduler.ExtenderConfig{
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]*scheduler.ExtenderConfig{
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
			[]*scheduler.ExtenderConfig{},
			[]*scheduler.ExtenderConfig{},
		},
		{
			"subset",
			false,
			[]*scheduler.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]*scheduler.ExtenderConfig{
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
			[]*scheduler.ExtenderConfig{},
			[]*scheduler.ExtenderConfig{
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
			[]*scheduler.ExtenderConfig{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:9000",
					FilterVerb: "get",
				},
			},
			[]*scheduler.ExtenderConfig{
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
