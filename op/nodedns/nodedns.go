package nodedns

import (
	"bytes"
	"text/template"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type unboundConfigTemplate struct {
	Domain    string
	ClusterIP string
	Upstreams []string
}

const unboundConfigTemplateText = `
server:
  do-daemonize: no
  interface: 0.0.0.0
  interface-automatic: yes
  access-control: 0.0.0.0/0 allow
  chroot: ""
  username: ""
  directory: "/etc/unbound"
  logfile: ""
  use-syslog: no
  log-time-ascii: yes
  log-queries: yes
  log-replies: yes
  log-local-actions: yes
  log-servfail: yes
  rrset-roundrobin: yes
  pidfile: "/tmp/unbound.pid"
  infra-host-ttl: 60
  prefetch: yes
remote-control:
  control-enable: yes
  control-interface: 127.0.0.1
  control-use-cert: no
stub-zone:
  name: "{{ .Domain }}"
  stub-addr: {{ .ClusterIP }}
forward-zone:
  name: "in-addr.arpa."
  forward-addr: {{ .ClusterIP }}
forward-zone:
  name: "ip6.arpa."
  forward-addr: {{ .ClusterIP }}
{{- if .Upstreams }}
forward-zone:
  name: "."
  {{- range .Upstreams }}
  forward-addr: {{ . }}
  {{- end }}
{{- end }}
`

// UnboundTemplateVersion is the version of unbound template
const UnboundTemplateVersion = "1"

var unboundDaemonSetText = `
metadata:
  name: node-dns
  namespace: kube-system
  annotations:
    cke.cybozu.com/image: ` + cke.UnboundImage.Name() + `
    cke.cybozu.com/template-version: ` + UnboundTemplateVersion + `
spec:
  selector:
    matchLabels:
      cke.cybozu.com/appname: node-dns
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: node-dns
    spec:
      priorityClassName: system-node-critical
      nodeSelector:
        beta.kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Make sure unbound gets scheduled on all nodes.
        - effect: NoSchedule
          operator: Exists
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      terminationGracePeriodSeconds: 0
      containers:
        - name: unbound
          image: ` + cke.UnboundImage.Name() + `
          args:
            - -c
            - /etc/unbound/unbound.conf
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              add:
              - NET_BIND_SERVICE
              drop:
              - all
            readOnlyRootFilesystem: true
          livenessProbe:
            tcpSocket:
              port: 53
              host: localhost
            periodSeconds: 1
            initialDelaySeconds: 1
            failureThreshold: 6
          volumeMounts:
            - name: config-volume
              mountPath: /etc/unbound
        - name: reload
          image: ` + cke.UnboundImage.Name() + `
          command:
          - /usr/local/bin/reload-unbound
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
              - all
            readOnlyRootFilesystem: true
          volumeMounts:
            - name: config-volume
              mountPath: /etc/unbound
      volumes:
        - name: config-volume
          configMap:
            name: node-dns
            items:
            - key: unbound.conf
              path: unbound.conf
`

// ConfigMap returns ConfigMap for unbound daemonset
func ConfigMap(clusterIP, domain string, dnsServers []string) *corev1.ConfigMap {
	var confTempl unboundConfigTemplate
	confTempl.Domain = domain
	confTempl.ClusterIP = clusterIP
	confTempl.Upstreams = dnsServers

	tmpl := template.Must(template.New("").Parse(unboundConfigTemplateText))
	unboundConf := new(bytes.Buffer)
	err := tmpl.Execute(unboundConf, confTempl)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      op.NodeDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"unbound.conf": unboundConf.String(),
		},
	}
}
