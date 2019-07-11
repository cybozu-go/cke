package cke

import (
	"testing"
)

func TestParseResource(t *testing.T) {
	cases := []struct {
		name        string
		yaml        string
		key         string
		expectError bool
	}{
		{
			"Namespace",
			`apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/instance: monitoring
  name: monitoring
`,
			"Namespace/monitoring",
			false,
		},
		{
			"ServiceAccount",
			`apiVersion: v1
kind: ServiceAccount
metadata:
  name: coil-controller
  namespace: kube-system
`,
			"ServiceAccount/kube-system/coil-controller",
			false,
		},
		{
			"ConfigMap",
			`kind: ConfigMap
apiVersion: v1
metadata:
  name: coil-config
  namespace: kube-system
data:
  cni_netconf: |-
    {
        "cniVersion": "0.3.1",
        "name": "k8s-pod-network"
    }
`,
			"ConfigMap/kube-system/coil-config",
			false,
		},
		{
			"Service",
			`kind: Service
apiVersion: v1
metadata:
  name: squid
  namespace: internet-egress
spec:
  type: NodePort
  selector:
    app.kubernetes.io/name: squid
  ports:
    - protocol: TCP
      nodePort: 30128
      port: 3128
      targetPort: 3128
`,
			"Service/internet-egress/squid",
			false,
		},
		{
			"PodSecurityPolicy",
			`apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: privileged
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: '*'
spec:
  privileged: true
  allowPrivilegeEscalation: true
  allowedCapabilities:
  - '*'
  volumes:
  - '*'
  hostNetwork: true
  hostPorts:
  - min: 0
    max: 65535
  hostIPC: true
  hostPID: true
  runAsUser:
    rule: 'RunAsAny'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
`,
			"PodSecurityPolicy/privileged",
			false,
		},
		{
			"NetworkPolicy",
			`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-network-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      role: db
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - ipBlock:
        cidr: 172.17.0.0/16
        except:
        - 172.17.1.0/24
    - namespaceSelector:
        matchLabels:
          project: myproject
    - podSelector:
        matchLabels:
          role: frontend
    ports:
    - protocol: TCP
      port: 6379
  egress:
  - to:
    - ipBlock:
        cidr: 10.0.0.0/24
    ports:
    - protocol: TCP
      port: 5978
`,
			"NetworkPolicy/default/test-network-policy",
			false,
		},
		{
			"ClusterNetworkPolicy",
			`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
spec:
  podSelector: {}
  policyTypes:
  - Ingress`,
			"NetworkPolicy//default-deny",
			false,
		},
		{
			"Role",
			`kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: default
  name: pod-reader
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "watch", "list"]`,
			"Role/default/pod-reader",
			false,
		},
		{
			"RoleV1Beta1",
			`kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  namespace: default
  name: pod-reader
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "watch", "list"]`,
			"Role/default/pod-reader",
			false,
		},
		{
			"RoleBinding",
			`kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-pods
  namespace: default
subjects:
- kind: User
  name: jane # Name is case sensitive
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role #this must be Role or ClusterRole
  name: pod-reader # this must match the name of the Role or ClusterRole you wish to bind to
  apiGroup: rbac.authorization.k8s.io`,
			"RoleBinding/default/read-pods",
			false,
		},
		{
			"ClusterRole",
			`kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: secret-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "watch", "list"]`,
			"ClusterRole/secret-reader",
			false,
		},
		{
			"ClusterRoleBinding",
			`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-secrets-global
subjects:
- kind: Group
  name: manager # Name is case sensitive
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: secret-reader
  apiGroup: rbac.authorization.k8s.io
`,
			"ClusterRoleBinding/read-secrets-global",
			false,
		},
		{
			"Deployment",
			`apiVersion: apps/v1
kind: Deployment
metadata:
  name: coil-controllers
  namespace: kube-system
  labels:
    app.kubernetes.io/name: coil-controllers
spec:
  # coil-controller can only have a single active instance.
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: coil-controllers
  strategy:
    type: Recreate
  template:
    metadata:
      name: coil-controllers
      namespace: kube-system
      labels:
        app.kubernetes.io/name: coil-controllers
    spec:
      priorityClassName: system-cluster-critical
      nodeSelector:
        kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
      serviceAccountName: coil-controller
      containers:
        - name: controller
          image: "%%COIL_IMAGE%%"
          command:
            - /coil-controller
            - "--etcd-endpoints=@cke-etcd"
            - "--etcd-tls-ca=/coil-secrets/etcd-ca.crt"
            - "--etcd-tls-cert=/coil-secrets/etcd-coil.crt"
            - "--etcd-tls-key=/coil-secrets/etcd-coil.key"
          # for "kubectl exec POD coilctl"
          env:
            - name: COILCTL_ENDPOINTS
              value: "@cke-etcd"
            - name: COILCTL_TLS_CA_FILE
              value: "/coil-secrets/etcd-ca.crt"
            - name: COILCTL_TLS_CERT_FILE
              value: "/coil-secrets/etcd-coil.crt"
            - name: COILCTL_TLS_KEY_FILE
              value: "/coil-secrets/etcd-coil.key"
          volumeMounts:
            - mountPath: /coil-secrets
              name: etcd-certs
      volumes:
        - name: etcd-certs
          secret:
            secretName: coil-etcd-secrets
            defaultMode: 0400
`,
			"Deployment/kube-system/coil-controllers",
			false,
		},
		{
			"DaemonSet",
			`kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: coil-node
  namespace: kube-system
  labels:
    app.kubernetes.io/name: coil-node
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: coil-node
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/name: coil-node
    spec:
      priorityClassName: system-node-critical
      nodeSelector:
        kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Make sure coil gets scheduled on all nodes.
        - effect: NoSchedule
          operator: Exists
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: coil-node
      terminationGracePeriodSeconds: 0
      containers:
        - name: coild
          image: "%%COIL_IMAGE%%"
          command:
            - /coild
            - "--etcd-endpoints=@cke-etcd"
            - "--etcd-tls-ca=/coil-secrets/etcd-ca.crt"
            - "--etcd-tls-cert=/coil-secrets/etcd-coil.crt"
            - "--etcd-tls-key=/coil-secrets/etcd-coil.key"
          env:
            - name: COIL_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          livenessProbe:
            httpGet:
              path: /status
              port: 9383
              host: localhost
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          volumeMounts:
            - mountPath: /coil-secrets
              name: etcd-certs
        # This container installs the coil CNI plugin and configuration file.
      volumes:
        # Used by installer.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        # Used by coild
        - name: etcd-certs
          secret:
            secretName: coil-etcd-secrets
            defaultMode: 0400
`,
			"DaemonSet/kube-system/coil-node",
			false,
		},
		{
			"CronJob",
			`kind: CronJob
apiVersion: batch/v2alpha1
metadata:
  name: etcd-backup
  namespace: kube-system
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: etcdbackup
              image: quay.io/cybozu/etcd:3.3.12
              command:
                - curl
                - -s
                - -XPOST
                - http://etcdbackup:8080/api/v1/backup
          restartPolicy: Never
  schedule:  "* 0 0 0 0"
`,
			"CronJob/kube-system/etcd-backup",
			false,
		},
		{
			"StatefulSet",
			`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  replicas: 3 # by default is 1
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx # has to match .spec.selector.matchLabels
    spec:
      terminationGracePeriodSeconds: 10
      containers:
      - name: nginx
        image: k8s.gcr.io/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "my-storage-class"
      resources:
        requests:
          storage: 1Gi`,
			"",
			true,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			key, data, err := ParseResource([]byte(c.yaml))
			if c.expectError {
				if err == nil {
					t.Error("error should have occurred")
				}
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			if key != c.key {
				t.Error("unexpected key: ", c.key, key)
			}

			t.Log(string(data))
		})
	}
}
