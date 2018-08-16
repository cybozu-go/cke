package cke

func controllerManagerKubeconfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: http://localhost:18080
users:
- name: controller-manager
contexts:
- context:
    cluster: local
    user: controller-manager
`
}

func schedulerKubeconfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: http://localhost:18080
users:
- name: controller-manager
contexts:
- context:
    cluster: local
    user: controller-manager
`
}
