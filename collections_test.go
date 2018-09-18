package cke

import "testing"

func TestCompareStrings(t *testing.T) {
	cases := []struct {
		s1 []string
		s2 []string
		ok bool
	}{
		{[]string{}, nil, true},
		{nil, []string{}, true},
		{nil, nil, true},
		{[]string{}, []string{}, true},
		{[]string{"hello", "world"}, []string{"hello", "world"}, true},

		{[]string{""}, []string{}, false},
		{[]string{"A"}, []string{"B"}, false},
	}
	for _, c := range cases {
		if compareStrings(c.s1, c.s2) != c.ok {
			t.Errorf("compareStrings(%#v, %#v) != %v", c.s1, c.s2, c.ok)
		}
	}
}

func TestCompareStringMap(t *testing.T) {
	cases := []struct {
		m1 map[string]string
		m2 map[string]string
		ok bool
	}{
		{map[string]string{}, nil, true},
		{nil, map[string]string{}, true},
		{nil, nil, true},
		{map[string]string{}, map[string]string{}, true},
		{map[string]string{"hello": "world"}, map[string]string{"hello": "world"}, true},

		{map[string]string{"hello": ""}, map[string]string{}, false},
		{map[string]string{"hello": "world"}, map[string]string{"good": "morning"}, false},
		{map[string]string{"hello": "world"}, map[string]string{"hello": "ola"}, false},
	}
	for _, c := range cases {
		if compareStringMap(c.m1, c.m2) != c.ok {
			t.Errorf("compareStringMap(%#v, %#v) != %v", c.m1, c.m2, c.ok)
		}
	}
}

func TestCompareMounts(t *testing.T) {
	cases := []struct {
		m1 []Mount
		m2 []Mount
		ok bool
	}{
		{[]Mount{}, nil, true},
		{nil, []Mount{}, true},
		{nil, nil, true},
		{[]Mount{}, []Mount{}, true},
		{[]Mount{{"/var", "/var", true, "", ""}}, []Mount{{"/var", "/var", true, "", ""}}, true},

		{[]Mount{{"/tmp", "/tmp", true, "", ""}}, []Mount{{"/var", "/var", true, "", ""}}, false},
	}
	for _, c := range cases {
		if compareMounts(c.m1, c.m2) != c.ok {
			t.Errorf("compareMounts(%#v, %#v) != %v", c.m1, c.m2, c.ok)
		}
	}
}
