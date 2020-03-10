package server

import (
	"testing"

	schedulerv1 "k8s.io/kube-scheduler/config/v1"
)

func TestEqualExtenderConfigs(t *testing.T) {
	testCases := []struct {
		message  string
		expected bool
		configs1 []schedulerv1.Extender
		configs2 []schedulerv1.Extender
	}{
		{
			"valid",
			true,
			[]schedulerv1.Extender{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
			[]schedulerv1.Extender{{
				URLPrefix:  "http://localhost",
				FilterVerb: "get",
			}},
		},
		{
			"if order is difference, it should return false",
			false,
			[]schedulerv1.Extender{
				{
					URLPrefix:  "http://localhost:8000",
					FilterVerb: "get",
				},
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]schedulerv1.Extender{
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
			[]schedulerv1.Extender{},
			[]schedulerv1.Extender{},
		},
		{
			"subset",
			false,
			[]schedulerv1.Extender{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
			},
			[]schedulerv1.Extender{
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
			[]schedulerv1.Extender{},
			[]schedulerv1.Extender{
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
			[]schedulerv1.Extender{
				{
					URLPrefix:  "http://localhost",
					FilterVerb: "post",
				},
				{
					URLPrefix:  "http://localhost:9000",
					FilterVerb: "get",
				},
			},
			[]schedulerv1.Extender{
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
		if equalExtenders(tc.configs1, tc.configs2) != tc.expected {
			t.Error(tc.message)
		}
	}
}
