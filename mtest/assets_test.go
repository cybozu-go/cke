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

//go:embed reboot-eviction-dry-run.yaml
var rebootEvictionDryRunYAML []byte

//go:embed reboot-slow-eviction-deployment.yaml
var rebootSlowEvictionDeploymentYAML []byte

//go:embed reboot-alittleslow-eviction-deployment.yaml
var rebootALittleSlowEvictionDeploymentYAML []byte

//go:embed repair-deployment.yaml
var repairDeploymentYAML []byte

//go:embed webhook-resources.yaml
var webhookYAML []byte

//go:embed trusted-rest-mapping-crd.yaml
var trustedRESTMappingCRDYAML []byte

//go:embed trusted-rest-mapping-cr.yaml
var trustedRESTMappingCRYAML []byte

//go:embed crd-test.yaml
var crdTestYAML []byte

//go:embed cr-test.yaml
var crTestYAML []byte
