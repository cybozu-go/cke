package mtest

import _ "embed"

//go:embed nginx.yml
var nginxYAML []byte

//go:embed reboot-deployment.yaml
var rebootDeploymentYAML []byte

//go:embed reboot-job-completed.yaml
var rebootJobCompletedYAML []byte

//go:embed reboot-job-running.yaml
var rebootJobRunningYAML []byte

//go:embed webhook-resources.yaml
var webhookYAML []byte
