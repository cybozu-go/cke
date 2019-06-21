Scheduler extender support
==========================

Overview
--------

CKE deploys resources which is needed to use a scheduler extender.

- [`k8s.io/component-base/config/v1alpha1.KubeSchedulerConfiguration`](https://github.com/kubernetes/kube-scheduler/blob/b74e9e79538d3a93ad1d1f391b9461c04a20c84e/config/v1alpha1/types.go#L38)
- [`k8s.io/kubernetes/pkg/scheduler/api/v1.Policy`](https://github.com/kubernetes/kubernetes/blob/release-1.14/pkg/scheduler/api/v1/types.go#L31)

CKE operates for scheduler extender as follows:
  1. Deploy `KubeSchedulerConfiguration` yaml and `Policy` json to the control plane.
  2. Deploy `Role` and `RoleBinding` to allow kube-scheduler to access `Endpoints` for leader election.
  3. Start kube-scheduler with step 1. created configuration files.

Caveats
-------

In `scheduler_conf`, `.algorithmSource.policy` must be `file`,
because CKE does not support `.algorithmSource.policy.configMap`.

Example
-------

```yaml
...
options:
  kube-scheduler:
    extenders:
      - name: topolvm-scheduler
        scheduler_conf: |
          apiVersion: kubescheduler.config.k8s.io/v1alpha1
          kind: KubeSchedulerConfiguration
          schedulerName: default-scheduler
          clientConnection:
            kubeconfig: "/etc/kubernetes/scheduler/kubeconfig"
          algorithmSource:
            policy:
              file:
                path: /etc/kubernetes/scheduler/topolvm-scheduler-policy.cfg
          leaderElection:
            leaderElect: true
            lockObjectName: topolvm-scheduler
            lockObjectNamespace: kube-system
        policy: |
          {
            "kind" : "Policy",
            "apiVersion" : "v1",
            "extenders" :
              [{
                "urlPrefix": "http://127.0.0.1:9251/",
                "filterVerb": "predicate",
                "prioritizeVerb": "prioritize",
                "nodeCacheCapable": false,
                "weight": 1,
                "managedResources":
                  [{
                    "name": "topolvm.cybozu.com/capacity",
                    "ignoredByScheduler": true
                  }]
              }]
           }
```
