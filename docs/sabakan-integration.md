Automatic cluster maintenance with sabakan
==========================================

[Sabakan][sabakan] is a network boot service.  It has a registry of machines in
a data center and keeps status information of machines.  It is convenient if CKE
can reference sabakan registry and generates Kubernetes cluster configuration
by itself.

In fact, CKE can be integrated with sabakan to achieve this.

How it works
------------

When enabled, CKE periodically query sabakan to retrieve available machines
in a data center and generates [cluster](cluster.md) configuration from a
user-supplied template.

Users can specify variables for the query to choose machines.  The query
will be executed by sabakan [GraphQL `searchMachines`](https://github.com/cybozu-go/sabakan/blob/master/docs/graphql.md) API.

Labels and other attributes of sabakan [`Machine`][schema] will be
translated into Kubernetes Node labels.

To keep Kubernetes and etcd cluster stable, CKE deliberately changes the
cluster configuration.  Details are described in the following sections.

When the configuration template is updated, CKE will soon regenerate
the cluster configuration from the new template.

Query and variables
-------------------

CKE uses the following GraphQL query to retrieve machine information from sabakan:

```
query ckeSearch($having: MachineParams = null,
                $notHaving: MachineParams = {
                  roles: ["boot"]
                  states: [RETIRED]
                }) {
  searchMachines(having: $having, notHaving: $notHaving) {
    # snip
  }
}
```

<a name="variables"></a>
Users may specify `$having` and `$notHaving` to change the search conditions.
They can be specified by a JSON object like this:

```json
{
  "having": {
    "labels": [{"name": "foo", "value": "bar"}],
    "roles": ["worker", "gpu"]
  },
  "notHaving": {
    "states": ["UNINITIALIZED", "RETIRED"]
  }
}
```

`$having` and `$notHaving` are `MachineParams`.  Consult [GraphQL schema][schema]
for the definition of `MachineParams`.

Strategy
--------

CKE generates cluster configuration with the following conditions.

*Musts*:

* User-specified [constraints](constraints.md) must be satisfied.
* etcd cluster must not be broken.
* Kubernetes cluster must not be broken.

*Shoulds*:

* Newer machines should be preferred than old ones.
* Healthy machines should be preferred than non-healthy ones.
* Unreachable machines should be removed from the cluster.  
    This is because CKE cannot work if the cluster configuration contains dead nodes.
* Unhealthy machines in the cluster should be [tainted][taint] with `NoSchedule`.
* Retiring machines should be [tainted][taint] with `NoExecute`.
* Rebooting machines should not be removed from the cluster nor be tainted.
* Each change of the cluster configuration should be made as small as possible.
* Control plane nodes should be distributed across different racks.
* All control plane nodes should be healthy.

To understand the status and lifecycle of a machine, see [sabakan lifecycle][lifecycle].

Algorithms
----------

### Node selection

When a new node need to be added to the cluster configuration, the algorithm
select a machine as follows:

1. Deselect non-healthy machines.
1. Deselect machines used in the current cluster configuration.
1. Add scores to each machine as follows:
    - If the machine's lifetime is > 250 days, +10.
    - If the machine's lifetime is > 500 days, +10 (+20 in total).
    - If the machine's lifetime is > 1000 days, +10 (+30 in total).
    - If the machine's lifetime is < -250 days, -10.
    - If the machine's lifetime is < -500 days, -10 (-20 in total).
    - If the machine's lifetime is < -1000 days, -10 (-30 in total).
    - If the cluster contains `n` machines in the same rack, - `min(n, 10)`.
1. Select the highest scored machine.

When an existing node need to be removed from the cluster configuration,
the algorithm select one as follows:

1. Add scores to each machine as follows:
    - If the machine's lifetime is > 250 days, +10.
    - If the machine's lifetime is > 500 days, +10 (+20 in total).
    - If the machine's lifetime is > 1000 days, +10 (+30 in total).
    - If the machine's lifetime is < -250 days, -10.
    - If the machine's lifetime is < -500 days, -10 (-20 in total).
    - If the machine's lifetime is < -1000 days, -10 (-30 in total).
    - If the machine is not healthy, -30.
    - If the cluster contains `n` machines in the same rack, - `min(n, 10)`.
1. Select the lowest scored machine.

Note that node selection should be done separately for control plane nodes
and non-control plane nodes.

### Initialization

The first time CKE generates cluster configuration from a template, it works
as follows:

1. Search sabakan to acquire the list of available machines.
1. Select `control-plane-count` nodes for Kubernetes/etcd control plane.
1. Select `minimum-workers` nodes.

The algorithm fails when available healthy nodes are not enough to
satisfy constraints.

### Maintenance

While CKE is idle, it queries sabakan periodically to check any updates on
the available machines.  The algorithm tries to minimize the change of the
cluster configuration.

First, CKE acquires _the list_ of available machines and its statuses from
sabakan then compares the list with _the current cluster_ configuration.

Then it selects one of the following actions if the condition matches.

#### Remove unreachable nodes

If the current cluster contains nodes that no longer exist in the list,
or if it contains unreachable nodes, they are removed.  This algorithm
should be chosen first as unreachable nodes block CKE.

New nodes may be added to satisfy constraints.

If too many control plane nodes would be removed, this algorithm cannot
work because replacing more than half of etcd servers would break the
cluster.  In this case, administrators need to fix the cluster
configuration manually.

#### Increase control plane nodes

When `control-plane-count` constraint is increased, control plane nodes are
added.  If there are too few unused machines and the number of worker nodes
are greater than `minimum-workers` constraint, existing healthy worker nodes
are *changed* to control plane nodes.

#### Decrease control plane nodes

When `control-plane-count` constraint is decreased, control plane nodes are
*changed* to non-control-plane nodes.

If the total number of worker nodes exceeds `maximum-workers`, existing
worker nodes are removed to satisfy the constraint.

#### Replace control nodes

If a control plane node is neither healthy, updating, or uninitialized,
the node is demoted to a worker, and a new machine is added as a control
plane node.

When there is no unused healthy machine, a healthy worker node is selected
to promote to a control plane node.

#### Increase worker nodes

If the number of healthy worker nodes is less than `minimum-workers` and
the total number of worker nodes is less than `maximum-workers`, new worker
nodes are added.

#### Decrease worker nodes

If a worker node is either retiring or retired for a while,

1. it is removed from the cluster if the number of workers is greater than `minimum-workers`, or
2. it is replaced with a new machine if available, otherwise,
3. it is left untouched.

#### Taint nodes

CKE adds  [taints][taint] to nodes as follows.
The taint key is `cke.cybozu.com/state`.

Machine state | Taint value | Taint effect
------------- | ----------- | ------------
Unhealthy     | `unhealthy` | `NoSchedule`
Retiring      | `retiring`  | `NoExecute`
Retired       | `retired`   | `NoExecute`

For other machine states, the taint is removed.

Node labels
-----------

Sabakan [`Machine`][schema] `labels` are translated to Kubernetes Node labels.
The label key will be prefixed by `sabakan.cke.cybozu.com/`.

Other Machine fields are also translated to labels as follows.

Field              | Label key                      | Value
------------------ | ------------------------------ | -----
`spec.rack`        | `cke.cybozu.com/rack`          | `spec.rack` converted to string.
`spec.indexInRack` | `cke.cybozu.com/index-in-rack` | `spec.indexInRack` converted to string.
`spec.role`        | `cke.cybozu.com/role`          | The same as `spec.role`.

Node annotations
----------------

Following Machine fields are translated to Node annotations:

Field               | Annotation key                 | Value
------------------- | ------------------------------ | -----
`spec.serial`       | `cke.cybozu.com/serial`        | The same as `spec.serial`.
`spec.registerDate` | `cke.cybozu.com/register-date` | `spec.registerDate` in RFC3339 format.
`spec.retireDate`   | `cke.cybozu.com/retire-date`   | `spec.retireDate` in RFC3339 format.


[sabakan]: https://github.com/cybozu-go/sabakan
[schema]: https://github.com/cybozu-go/sabakan/blob/master/gql/schema.graphql
[taint]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
[lifecycle]: https://github.com/cybozu-go/sabakan/blob/master/docs/lifecycle.md#transition-diagram
