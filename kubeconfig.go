package cke

func controllerManagerKubeconfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    certificate-authority: /etc/kubernetes/pki/ca.crt
    server: https://localhost:18080
users:
- name: controller-manager
- name: admin
  user:
    client-certificate: /etc/kubernetes/pki/admin.crt
    client-key: /etc/kubernetes/pki/admin.key
contexts:
- context:
    cluster: local
    user: admin
`
}

func schedulerKubeconfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: https://localhost:18080
    certificate-authority: /etc/kubernetes/pki/ca.crt
users:
- name: controller-manager
- name: admin
  user:
    client-certificate: /etc/kubernetes/pki/admin.crt
    client-key: /etc/kubernetes/pki/admin.key
contexts:
- context:
    cluster: local
    user: admin
`
}

func kubeletKubeConfig() string {
	return `apiVersion: v1
clusters:
- name: local
  cluster:
    server: https://localhost:18080
    certificate-authority: /etc/kubernetes/pki/ca.crt
users:
- name: kubelet
- name: admin
  user:
    client-certificate: /etc/kubernetes/pki/admin.crt
    client-key: /etc/kubernetes/pki/admin.key
contexts:
- context:
    cluster: local
    user: admin
`
}
