package cke

import "testing"

func TestParseResource(t *testing.T) {
	t.Parallel()

	y := `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/instance: monitoring
  name: monitoring
`

	key, jd, err := ParseResource([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if key != "Namespace/monitoring" {
		t.Error(`key != "Namespace/monitoring":`, key)
	}
	t.Log(string(jd))
}
