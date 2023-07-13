package mtest

import _ "embed"

//go:embed httpd.yml
var httpdYAML []byte

//go:embed reboot-deployment.yaml
var rebootDeploymentYAML []byte

//go:embed reboot-job-completed.yaml
var rebootJobCompletedYAML []byte

//go:embed reboot-job-running.yaml
var rebootJobRunningYAML []byte

//go:embed reboot-slow-eviction-deployment.yaml
var rebootSlowEvictionDeploymentYAML []byte

//go:embed reboot-alittleslow-eviction-deployment.yaml
var rebootALittleSlowEvictionDeploymentYAML []byte

//go:embed webhook-resources.yaml
var webhookYAML []byte
