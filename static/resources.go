// Code generated by compile_resources. DO NOT EDIT.
//go:generate go run ../pkg/compile_resources

package static

import (
	"github.com/cybozu-go/cke"
)

// Resources is the Kubernetes resource definitions embedded in CKE.
var Resources = []cke.ResourceDefinition{
	{
		Key:        "ServiceAccount/kube-system/cke-cluster-dns",
		Kind:       "ServiceAccount",
		Namespace:  "kube-system",
		Name:       "cke-cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: cke-cluster-dns\n  namespace: kube-system\n  annotations:\n    cke.cybozu.com/revision: \"1\"\n"),
	},
	{
		Key:        "ClusterRole/system:cluster-dns",
		Kind:       "ClusterRole",
		Namespace:  "",
		Name:       "system:cluster-dns",
		Revision:   2,
		Image:      "",
		Definition: []byte("\nkind: ClusterRole\napiVersion: rbac.authorization.k8s.io/v1\nmetadata:\n  name: system:cluster-dns\n  labels:\n    kubernetes.io/bootstrapping: rbac-defaults\n  annotations:\n    cke.cybozu.com/revision: \"2\"\n    # turn on auto-reconciliation\n    # https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation\n    rbac.authorization.kubernetes.io/autoupdate: \"true\"\nrules:\n  - apiGroups:\n      - \"\"\n    resources:\n      - services\n      - pods\n      - namespaces\n    verbs:\n      - list\n      - watch\n  - apiGroups:\n      - discovery.k8s.io\n    resources:\n      - endpointslices\n    verbs:\n      - list\n      - watch\n"),
	},
	{
		Key:        "ClusterRole/system:kube-apiserver-to-kubelet",
		Kind:       "ClusterRole",
		Namespace:  "",
		Name:       "system:kube-apiserver-to-kubelet",
		Revision:   1,
		Image:      "",
		Definition: []byte("kind: ClusterRole\napiVersion: rbac.authorization.k8s.io/v1\nmetadata:\n  name: system:kube-apiserver-to-kubelet\n  labels:\n    kubernetes.io/bootstrapping: rbac-defaults\n  annotations:\n    cke.cybozu.com/revision: \"1\"\n    # turn on auto-reconciliation\n    # https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation\n    rbac.authorization.kubernetes.io/autoupdate: \"true\"\nrules:\n  - apiGroups: [\"\"]\n    resources:\n      - nodes/proxy\n      - nodes/stats\n      - nodes/log\n      - nodes/spec\n      - nodes/metrics\n    verbs: [\"*\"]\n"),
	},
	{
		Key:        "ClusterRoleBinding/system:cluster-dns",
		Kind:       "ClusterRoleBinding",
		Namespace:  "",
		Name:       "system:cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("\nkind: ClusterRoleBinding\napiVersion: rbac.authorization.k8s.io/v1\nmetadata:\n  name: system:cluster-dns\n  labels:\n    kubernetes.io/bootstrapping: rbac-defaults\n  annotations:\n    cke.cybozu.com/revision: \"1\"\n    rbac.authorization.kubernetes.io/autoupdate: \"true\"\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: system:cluster-dns\nsubjects:\n- kind: ServiceAccount\n  name: cke-cluster-dns\n  namespace: kube-system\n"),
	},
	{
		Key:        "ClusterRoleBinding/system:kube-apiserver",
		Kind:       "ClusterRoleBinding",
		Namespace:  "",
		Name:       "system:kube-apiserver",
		Revision:   1,
		Image:      "",
		Definition: []byte("kind: ClusterRoleBinding\napiVersion: rbac.authorization.k8s.io/v1\nmetadata:\n  name: system:kube-apiserver\n  labels:\n    kubernetes.io/bootstrapping: rbac-defaults\n  annotations:\n    cke.cybozu.com/revision: \"1\"\n    rbac.authorization.kubernetes.io/autoupdate: \"true\"\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: system:kube-apiserver-to-kubelet\nsubjects:\n- kind: User\n  name: kubernetes\n"),
	},
	{
		Key:        "DaemonSet/kube-system/node-dns",
		Kind:       "DaemonSet",
		Namespace:  "kube-system",
		Name:       "node-dns",
		Revision:   4,
		Image:      "ghcr.io/cybozu/unbound:1.23.0.1,ghcr.io/cybozu/unbound_exporter:0.4.6.3",
		Definition: []byte("kind: DaemonSet\napiVersion: apps/v1\nmetadata:\n  name: node-dns\n  namespace: kube-system\n  annotations:\n    cke.cybozu.com/image: \"ghcr.io/cybozu/unbound:1.23.0.1,ghcr.io/cybozu/unbound_exporter:0.4.6.3\"\n    cke.cybozu.com/revision: \"4\"\nspec:\n  selector:\n    matchLabels:\n      cke.cybozu.com/appname: node-dns\n  updateStrategy:\n    type: RollingUpdate\n    rollingUpdate:\n      maxSurge: 35%\n      maxUnavailable: 0\n  template:\n    metadata:\n      labels:\n        cke.cybozu.com/appname: node-dns\n    spec:\n      priorityClassName: system-node-critical\n      nodeSelector:\n        kubernetes.io/os: linux\n      hostNetwork: true\n      tolerations:\n        - operator: Exists\n      terminationGracePeriodSeconds: 1\n      containers:\n        - name: unbound\n          image: ghcr.io/cybozu/unbound:1.23.0.1\n          args:\n            - -c\n            - /etc/unbound/unbound.conf\n          securityContext:\n            allowPrivilegeEscalation: false\n            capabilities:\n              add:\n              - NET_BIND_SERVICE\n              drop:\n              - all\n            readOnlyRootFilesystem: true\n          readinessProbe:\n            tcpSocket:\n              port: 53\n              host: localhost\n            periodSeconds: 1\n          livenessProbe:\n            tcpSocket:\n              port: 53\n              host: localhost\n            periodSeconds: 1\n            initialDelaySeconds: 1\n            failureThreshold: 6\n          volumeMounts:\n            - name: config-volume\n              mountPath: /etc/unbound\n            - name: var-run-unbound\n              mountPath: /var/run/unbound\n          resources:\n            requests:\n              cpu: 50m\n              memory: 250Mi\n        - name: reload\n          image: ghcr.io/cybozu/unbound:1.23.0.1\n          command:\n          - /usr/local/bin/reload-unbound\n          securityContext:\n            allowPrivilegeEscalation: false\n            capabilities:\n              drop:\n              - all\n            readOnlyRootFilesystem: true\n          volumeMounts:\n            - name: config-volume\n              mountPath: /etc/unbound\n            - name: var-run-unbound\n              mountPath: /var/run/unbound\n        - name: exporter\n          image: ghcr.io/cybozu/unbound_exporter:0.4.6.3\n          args:\n          # must be same with the path written in /op/nodedns/nodedns.go\n          - --unbound.host=unix:///var/run/unbound/unbound.sock\n          - --web.reuse-port=true\n          securityContext:\n            allowPrivilegeEscalation: false\n            capabilities:\n              drop:\n              - all\n            readOnlyRootFilesystem: true\n          volumeMounts:\n            - name: var-run-unbound\n              mountPath: /var/run/unbound\n      volumes:\n        - name: config-volume\n          configMap:\n            name: node-dns\n            items:\n            - key: unbound.conf\n              path: unbound.conf\n        - name: var-run-unbound\n          emptyDir: {}\n"),
	},
	{
		Key:        "Deployment/kube-system/cluster-dns",
		Kind:       "Deployment",
		Namespace:  "kube-system",
		Name:       "cluster-dns",
		Revision:   4,
		Image:      "ghcr.io/cybozu/coredns:1.12.2.1",
		Definition: []byte("\nkind: Deployment\napiVersion: apps/v1\nmetadata:\n  name: cluster-dns\n  namespace: kube-system\n  annotations:\n    cke.cybozu.com/image: \"ghcr.io/cybozu/coredns:1.12.2.1\"\n    cke.cybozu.com/revision: \"4\"\nspec:\n  replicas: 2\n  strategy:\n    type: RollingUpdate\n    rollingUpdate:\n      maxUnavailable: 1\n  selector:\n    matchLabels:\n      cke.cybozu.com/appname: cluster-dns\n  template:\n    metadata:\n      labels:\n        cke.cybozu.com/appname: cluster-dns\n        k8s-app: coredns # sonobuoy requires\n      annotations:\n        prometheus.io/port: \"9153\"\n    spec:\n      priorityClassName: system-cluster-critical\n      serviceAccountName: cke-cluster-dns\n      tolerations:\n        - key: node-role.kubernetes.io/master\n          effect: NoSchedule\n        - key: \"CriticalAddonsOnly\"\n          operator: \"Exists\"\n        - key: kubernetes.io/e2e-evict-taint-key\n          operator: Exists\n          # for sonobuoy https://github.com/vmware-tanzu/sonobuoy/pull/878\n      containers:\n      - name: coredns\n        image: ghcr.io/cybozu/coredns:1.12.2.1\n        imagePullPolicy: IfNotPresent\n        resources:\n          requests:\n            cpu: 50m\n            memory: 250Mi\n        args: [ \"-conf\", \"/etc/coredns/Corefile\" ]\n        lifecycle:\n          preStop:\n            exec:\n              command: [\"sh\", \"-c\", \"sleep 5\"]\n        volumeMounts:\n        - name: config-volume\n          mountPath: /etc/coredns\n          readOnly: true\n        ports:\n        - containerPort: 1053\n          name: dns\n          protocol: UDP\n        - containerPort: 1053\n          name: dns-tcp\n          protocol: TCP\n        - containerPort: 9153\n          name: metrics\n          protocol: TCP\n        securityContext:\n          allowPrivilegeEscalation: false\n          capabilities:\n            drop:\n            - all\n          readOnlyRootFilesystem: true\n        readinessProbe:\n          httpGet:\n            path: /ready\n            port: 8181\n            scheme: HTTP\n        livenessProbe:\n          httpGet:\n            path: /health\n            port: 8080\n            scheme: HTTP\n          initialDelaySeconds: 60\n          timeoutSeconds: 5\n          successThreshold: 1\n          failureThreshold: 5\n      dnsPolicy: Default\n      volumes:\n        - name: config-volume\n          configMap:\n            name: cluster-dns\n            items:\n            - key: Corefile\n              path: Corefile\n      affinity:\n        podAntiAffinity:\n          requiredDuringSchedulingIgnoredDuringExecution:\n          - labelSelector:\n              matchLabels:\n                cke.cybozu.com/appname: cluster-dns\n            topologyKey: \"kubernetes.io/hostname\"\n"),
	},
	{
		Key:        "PodDisruptionBudget/kube-system/cluster-dns-pdb",
		Kind:       "PodDisruptionBudget",
		Namespace:  "kube-system",
		Name:       "cluster-dns-pdb",
		Revision:   1,
		Image:      "",
		Definition: []byte("\napiVersion: policy/v1\nkind: PodDisruptionBudget\nmetadata:\n  name: cluster-dns-pdb\n  namespace: kube-system\n  annotations:\n    cke.cybozu.com/revision: \"1\"\nspec:\n  maxUnavailable: 1\n  selector:\n    matchLabels:\n      cke.cybozu.com/appname: cluster-dns\n"),
	},
	{
		Key:        "Service/kube-system/cluster-dns",
		Kind:       "Service",
		Namespace:  "kube-system",
		Name:       "cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("\nkind: Service\napiVersion: v1\nmetadata:\n  name: cluster-dns\n  namespace: kube-system\n  annotations:\n    cke.cybozu.com/revision: \"1\"\n  labels:\n    cke.cybozu.com/appname: cluster-dns\nspec:\n  selector:\n    cke.cybozu.com/appname: cluster-dns\n  ports:\n    - name: dns\n      port: 53\n      targetPort: 1053\n      protocol: UDP\n    - name: dns-tcp\n      port: 53\n      targetPort: 1053\n      protocol: TCP\n"),
	},
}
