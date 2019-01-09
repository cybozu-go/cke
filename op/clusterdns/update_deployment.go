package clusterdns

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// retrieved from https://github.com/kelseyhightower/kubernetes-the-hard-way
var deploymentText = `
metadata:
  name: cluster-dns
  namespace: kube-system
  annotations:
    cke.cybozu.com/image: ` + cke.CoreDNSImage.Name() + `
    cke.cybozu.com/template-version: ` + CoreDNSTemplateVersion + `
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      cke.cybozu.com/appname: cluster-dns
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: cluster-dns
    spec:
      priorityClassName: system-cluster-critical
      serviceAccountName: cluster-dns
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
      - name: coredns
        image: ` + cke.CoreDNSImage.Name() + `
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - all
          readOnlyRootFilesystem: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: ` + op.ClusterDNSAppName + `
            items:
            - key: Corefile
              path: Corefile
`

type updateDeploymentOp struct {
	apiserver *cke.Node
	finished  bool
}

// UpdateDeploymentOp returns an Operator to update deployment of CoreDNS.
func UpdateDeploymentOp(apiserver *cke.Node) cke.Operator {
	return &updateDeploymentOp{
		apiserver: apiserver,
	}
}

func (o *updateDeploymentOp) Name() string {
	return "update-cluster-dns-deployment"
}

func (o *updateDeploymentOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateDeploymentCommand{o.apiserver}
}

type updateDeploymentCommand struct {
	apiserver *cke.Node
}

func (c updateDeploymentCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	deployment := new(v1.Deployment)
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(deployment)
	if err != nil {
		return err
	}
	_, err = deployments.Update(deployment)
	return err
}

func (c updateDeploymentCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateDeploymentCommand",
		Target: "kube-system",
	}
}
