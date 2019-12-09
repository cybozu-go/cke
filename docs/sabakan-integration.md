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

Cluster template
----------------

Sabakan integration generates the cluster definition from a user-defined template.
The template syntax is the same as [cluster.yml](cluster.md).

The difference is that the template must have at least one control plane node
and one non-control plane node.

A minimal template looks like:

```yaml
name: cluster
nodes:
- user: cybozu
  control_plane: true
- user: cybozu
  control_plane: false
service_subnet: 10.68.0.0/16
pod_subnet: 10.64.0.0/14
```

When the configuration template is updated, CKE will soon regenerate
the cluster configuration from the new template.

Roles and weights
-----------------

In general, servers in data centers can be classified into several types.
For example, _compute_ servers are typically used to run VM or container workloads
while _storage_ servers are used to store large persistent data.

Servers defined in sabakan have a mandatory attribute "role" for this classification.

To support a Kubernetes cluster consisting of multiple roles such as
"compute", "storage", and "gpu", sabakan integration of CKE allows users to
specify node templates for each server role and its weight (ratio) in the
cluster.

The node role is specified with `cke.cybozu.com/role` label value.
The weight (ratio) in the cluster is specified with `cke.cybozu.com/weight` label value.

An example to specify the ratio between compute/storage/gpu to 6:3:1 looks like:

```yaml
name: cluster
nodes:
- user: cybozu
  control_plane: true
  labels:
    # Use compute servers for control plane nodes
    cke.cybozu.com/role: "compute"
- user: cybozu
  labels:
    cke.cybozu.com/role: "compute"
    cke.cybozu.com/weight: "6.0"
- user: cybozu
  labels:
    cke.cybozu.com/role: "storage"
    cke.cybozu.com/weight: "3.0"
  taints:
  - key: cke.cybozu.com/role
    value: storage
    effect: NoExecute
- user: cybozu
  labels:
    cke.cybozu.com/role: "gpu"
    cke.cybozu.com/weight: "1.0"
  taints:
  - key: cke.cybozu.com/role
    value: gpu
    effect: PreferNoSchedule
service_subnet: 10.68.0.0/16
pod_subnet: 10.64.0.0/14
```

### Node template without role

If a node template lacks `cke.cybozu.com/role` label, any servers can match it.

If there are more than two templates for non-control plane nodes, they must have
`cke.cybozu.com/role` label.

### Node template without weight

If a non-control plane node template lacks `cke.cybozu.com/weight` label, its weight becomes "1.0".

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

* If the template node for control plane has `cke.cybozu.com/role` label,
    servers of control plane nodes should be the specified role.
* The number of servers for each role should be proportional to the given weights in the template.
* Newer machines should be preferred than old ones.
* Healthy machines should be preferred than non-healthy ones.
* Unhealthy and unreachable machines in the cluster should be [tainted][taint] with `NoSchedule`.
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
2. Deselect machines used in the current cluster configuration.
3. Select machines of preferred roles.
    - If the new node will be a control plane and a role is specified, select servers of the same role.
    - Otherwise, choose a node template with the smallest number of servers than the specified weight,
        and use the role specified for the template.
4. Add scores to each machine as follows:
    - If the machine's lifetime is > 250 days, +10.
    - If the machine's lifetime is > 500 days, +10 (+20 in total).
    - If the machine's lifetime is > 1000 days, +10 (+30 in total).
    - If the machine's lifetime is < -250 days, -10.
    - If the machine's lifetime is < -500 days, -10 (-20 in total).
    - If the machine's lifetime is < -1000 days, -10 (-30 in total).
    - If the cluster contains `n` machines in the same rack, - `min(n, 10)`.
5. Select the highest scored machine.

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
2. Select the lowest scored machine.

Note that node selection should be done separately for control plane nodes
and non-control plane nodes.

### Initialization

The first time CKE generates cluster configuration from a template, it works
as follows:

1. Search sabakan to acquire the list of available machines.
2. Select `control-plane-count` nodes for Kubernetes/etcd control plane.
3. Select `minimum-workers` nodes.

The algorithm fails when available healthy nodes are not enough to
satisfy constraints.

### Maintenance

While CKE is idle, it queries sabakan periodically to check any updates on
the available machines.  The algorithm tries to minimize the change of the
cluster configuration.

First, CKE acquires _the list_ of available machines and its statuses from
sabakan then compares the list with _the current cluster_ configuration.

Then it selects one of the following actions if the condition matches.

#### Remove non-existent nodes

If the current cluster contains nodes that no longer exist in the list,
they are removed. This should be executed at the beginning because non-existent
nodes block CKE.

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

| Machine state | Taint value   | Taint effect |
| ------------- | ------------- | ------------ |
| Unhealthy     | `unhealthy`   | `NoSchedule` |
| Unreachable   | `unreachable` | `NoSchedule` |
| Retiring      | `retiring`    | `NoExecute`  |
| Retired       | `retired`     | `NoExecute`  |

For other machine states, the taint is removed.

Static Node Labels and Annotations
----------------------------------

#### Labels

Sabakan [`Machine`][schema] `labels` are translated to Kubernetes Node labels.
The label key will be prefixed by `sabakan.cke.cybozu.com/`.

Other Machine fields are also translated to labels as follows.

| Field              | Label key                      | Value                                   |
| ------------------ | ------------------------------ | --------------------------------------- |
| `spec.rack`        | `cke.cybozu.com/rack`          | `spec.rack` converted to string.        |
| `spec.indexInRack` | `cke.cybozu.com/index-in-rack` | `spec.indexInRack` converted to string. |
| `spec.role`        | `cke.cybozu.com/role`          | The same as `spec.role`.                |

#### Annotations

Following Machine fields are translated to Node annotations:

| Field               | Annotation key                 | Value                                  |
| ------------------- | ------------------------------ | -------------------------------------- |
| `spec.serial`       | `cke.cybozu.com/serial`        | The same as `spec.serial`.             |
| `spec.registerDate` | `cke.cybozu.com/register-date` | `spec.registerDate` in RFC3339 format. |
| `spec.retireDate`   | `cke.cybozu.com/retire-date`   | `spec.retireDate` in RFC3339 format.   |

Dynamic Node Labels and Annotations
-----------------------------------

CKE can also create labels in combination with fields under Sabakan [`Machine`][schema] `Spec`.
Only values of the labels are created with go-template (e.g. `rack: {{ .Rack }}` ).
The available fields can be referenced in [Labels](#labels) and [Annotations](#annotations).


[sabakan]: https://github.com/cybozu-go/sabakan
[schema]: https://github.com/cybozu-go/sabakan/blob/master/gql/schema.graphql
[taint]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
[lifecycle]: https://github.com/cybozu-go/sabakan/blob/master/docs/lifecycle.md#transition-diagram
