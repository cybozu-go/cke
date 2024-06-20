Automatic repair triggered by sabakan
=====================================

[Sabakan][sabakan] is management software for server machines in a data center.
It stores the status information of machines as well as their spec information.
By referring to machines' status information in sabakan, CKE can initiate the repair of a non-healthy machine.

This functionality is similar to [sabakan integration](sabakan-integration.md).

How it works
------------

CKE periodically queries sabakan to retrieve machines' status information in a data center.
If CKE finds non-healthy machines, it creates [repair queue entries](repair.md) for those machines.

The fields of a repair queue entry are determined based on the [information of the non-healthy machine](https://github.com/cybozu-go/sabakan/blob/main/docs/machine.md).
* `address`: `.spec.ipv4[0]`
* `machine_type`: `.spec.bmc.type`
* `operation`: `.status.state`

Users can configure the query to choose non-healthy machines.
The queries are executed via sabakan [GraphQL `searchMachines`](https://github.com/cybozu-go/sabakan/blob/master/docs/graphql.md) API.

Query
-----

CKE uses the following GraphQL query to retrieve machine information from sabakan.

```
query ckeSearch($having: MachineParams, $notHaving: MachineParams) {
  searchMachines(having: $having, notHaving: $notHaving) {
    # snip
  }
}
```

The following values are used for `$having` and `$notHaving` variables by default.
Users can change these values by [specifying a JSON object](ckecli.md#ckecli-auto-repair-set-variables-file).

```json
{
  "having": {
    "states": ["UNHEALTHY", "UNREACHABLE"]
  },
  "notHaving": {
    "roles": ["boot"]
  }
}
```

The type of `$having` and `$notHaving` is `MachineParams`.
Consult [GraphQL schema][schema] for the definition of `MachineParams`.

Enqueue limiters
----------------

### Limiter for a single machine

In order not to repeat repair operations too quickly for a single unstable machine, CKE checks recent repair queue entries before enqueueing.
If it finds a recent entry for the machine in question, no matter whether the entry has finished or not, it refrains from creating an additional entry.

CKE considers all persisting queue entries as "recent" for simplicity.
A user should delete a finished repair queue entry for a machine once they consider the machine repaired.
* If a repair queue entry has finished with success and a user considers the machine stable, they should delete the finished entry.
* If a repair queue entry has finished with failure or a user considers the machine unstable, they should repair the machine manually. After the machine gets repaired, they should delete the finished entry.

### Limiter for a cluster

Sabakan may occasionally report false-positive non-healthy machines.
If CKE believes all of the failure reports and initiates a lot of repair operations, the Kubernetes cluster will be stuck -- or worse, corrupted.

Even when the failure reports are correct, it would be good for CKE to refrain from repairing too many machines.
For example, the failure of many servers might be caused by the temporary power failure of a whole server rack.
In that case, CKE should not mark the machines unrepairable as a result of pointless repair operations.
Once the machines are marked unrepairable, sabakan will delete all data on those machines.

In order not to initiate too many repair operations, CKE checks the number of recent repair queue entries plus the number of new failure reports before enqueueing.
If it finds excessive numbers of entries/reports, no matter whether the entries have finished or not, it refrains from creating an additional entry.

The maximum number of recent repair queue entries and new failure reports is [configurable](ckecli.md#ckecli-constraints-set-name-value) as a [constraint `maximum-repair-queue-entries`](constraints.md).

As stated above, CKE considers all persisting queue entries as "recent" for simplicity.

### Limiter for planned reboot

A machine may become "UNREACHABLE" very quickly even if it is [being rebooted in a planned manner](reboot.md).
CKE should wait for a while before starting repair operations for a rebooting machine.

A user can [configure the wait time](ckecli.md#ckecli-constraints-set-name-value) as a [constraint `wait-seconds-to-repair-rebooting`](constraints.md).

CKE does not manage the reboot operations of out-of-cluster machines.
It cannot distinguish between the reboot and the crash of an out-of-cluster machine.
To avoid filling the repair queue with unnecessary entries, CKE waits for a while also before repairing an out-of-cluster machine.

[sabakan]: https://github.com/cybozu-go/sabakan
[schema]: https://github.com/cybozu-go/sabakan/blob/main/gql/graph/schema.graphqls
