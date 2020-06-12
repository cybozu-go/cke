Scheduler policy support
========================

Overview
--------

CKE deploys resources which is needed to use a scheduler policy.

- [`k8s.io/component-base/config/v1alpha1.KubeSchedulerConfiguration`](https://github.com/kubernetes/kube-scheduler/blob/b74e9e79538d3a93ad1d1f391b9461c04a20c84e/config/v1alpha1/types.go#L38)
- [`k8s.io/kubernetes/pkg/scheduler/api/v1.Policy`](https://github.com/kubernetes/kubernetes/blob/release-1.14/pkg/scheduler/api/v1/types.go#L31)

CKE operates for scheduler policy as follows:
  1. Using values of options.kube-scheduler.extenders, predicates, and priorities, CKE put `KubeSchedulerConfiguration` yaml and `Policy` json on the control planes' hosts.
     - For each config, `null` is put if the value is empty or not set.
  2. Deploy `Role` and `RoleBinding` to allow kube-scheduler to access `Endpoints` for leader election.
  3. Start kube-scheduler with step 1. created configuration files.

Example
-------

```yaml
...
options:
  kube-scheduler:
    extenders:
      - |
          {
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
          }
    priorities:
      - |
          {
            "name": "NodeAffinityPriority",
            "weight": 1
          }
      - |
          {
            "name": "EvenPodsSpreadPriority",
            "weight": 2
          }
```

output policy file

```json
{
  "apiVersion": "v1",
  "kind": "Policy",
	"extenders": [
    {
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
    }
  ],
  "predicates": null,
  "priorities": [
    {
      "name": "NodeAffinityPriority",
      "weight": 1
    }, {
      "name": "EvenPodsSpreadPriority",
      "weight": 2
    }
  ]
}
```
