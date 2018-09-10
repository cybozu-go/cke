package cke

func controllerManagerKubeconfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: https://localhost:16443
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
    server: https://localhost:16443
users:
- name: controller-manager
contexts:
- context:
    cluster: local
    user: controller-manager
`
}

func kubeletKubeConfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: https://localhost:16443
users:
- name: kubelet
contexts:
- context:
    cluster: local
    user: kubelet
`
}

func proxyKubeConfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: https://localhost:16443
users:
- name: kube-proxy
contexts:
- context:
    cluster: local
    user: kube-proxy
`
}
