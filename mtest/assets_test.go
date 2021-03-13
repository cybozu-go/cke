package mtest

import _ "embed"

//go:embed nginx.yml
var nginxYAML []byte

//go:embed mtest-policy.yml
var policyYAML []byte

//go:embed reboot-deployment.yaml
var rebootYAML []byte

//go:embed webhook-resources.yaml
var webhookYAML []byte
