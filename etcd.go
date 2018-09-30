package cke

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

const (
	defaultEtcdVolumeName = "etcd-cke"
	etcdContainerName     = "etcd"
	defaultEtcdTimeout    = 5 * time.Second
)

func etcdVolumeName(e EtcdParams) string {
	if len(e.VolumeName) == 0 {
		return defaultEtcdVolumeName
	}
	return e.VolumeName
}

func etcdEndpoints(nodes []*Node) []string {
	endpoints := make([]string, len(nodes))
	for i, n := range nodes {
		endpoints[i] = "https://" + n.Address + ":2379"
	}
	return endpoints
}

func addressInURLs(address string, urls []string) (bool, error) {
	for _, urlStr := range urls {
		u, err := url.Parse(urlStr)
		if err != nil {
			return false, err
		}
		h, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			return false, err
		}
		if h == address {
			return true, nil
		}
	}
	return false, nil
}

func etcdGuessMemberName(m *etcdserverpb.Member) (string, error) {
	if len(m.Name) > 0 {
		return m.Name, nil
	}

	if len(m.PeerURLs) == 0 {
		return "", errors.New("empty PeerURLs")
	}

	u, err := url.Parse(m.PeerURLs[0])
	if err != nil {
		return "", err
	}
	h, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return h, nil
}

func etcdBuiltInParams(node *Node, initialCluster []string, state string) ServiceParams {
	// NOTE: "--initial-*" flags and its value must be joined with '=' to
	// compare parameters to detect outdated parameters.
	args := []string{
		"--name=" + node.Address,
		"--listen-peer-urls=https://0.0.0.0:2380",
		"--listen-client-urls=https://0.0.0.0:2379",
		"--advertise-client-urls=https://" + node.Address + ":2379",
		"--cert-file=" + EtcdPKIPath("server.crt"),
		"--key-file=" + EtcdPKIPath("server.key"),
		"--client-cert-auth=true",
		"--trusted-ca-file=" + EtcdPKIPath("ca-client.crt"),
		"--peer-cert-file=" + EtcdPKIPath("peer.crt"),
		"--peer-key-file=" + EtcdPKIPath("peer.key"),
		"--peer-client-cert-auth=true",
		"--peer-trusted-ca-file=" + EtcdPKIPath("ca-peer.crt"),
		"--enable-v2=false",
		"--enable-pprof=true",
		"--auto-compaction-mode=periodic",
		"--auto-compaction-retention=24",
	}
	if len(initialCluster) > 0 {
		args = append(args,
			"--initial-advertise-peer-urls=https://"+node.Address+":2380",
			"--initial-cluster="+strings.Join(initialCluster, ","),
			"--initial-cluster-token=cke",
			"--initial-cluster-state="+state)
	}
	binds := []Mount{
		{
			Source:      "/etc/etcd/pki",
			Destination: "/etc/etcd/pki",
			ReadOnly:    true,
			Label:       LabelPrivate,
		},
	}
	params := ServiceParams{
		ExtraArguments: args,
		ExtraBinds:     binds,
	}

	return params
}

type etcdBootOp struct {
	endpoints []string
	nodes     []*Node
	params    EtcdParams
	step      int
	makeFiles *makeFilesCommand
}

// EtcdBootOp returns an Operator to bootstrap etcd cluster.
func EtcdBootOp(nodes []*Node, params EtcdParams) Operator {
	return &etcdBootOp{
		endpoints: etcdEndpoints(nodes),
		nodes:     nodes,
		params:    params,
		makeFiles: &makeFilesCommand{nodes: nodes},
	}
}

func (o *etcdBootOp) Name() string {
	return "etcd-bootstrap"
}

func (o *etcdBootOp) NextCommand() Commander {
	volname := etcdVolumeName(o.params)

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, EtcdImage}
	case 1:
		o.step++
		return prepareEtcdCertificatesCommand{o.makeFiles}
	case 2:
		o.step++
		return o.makeFiles
	case 3:
		o.step++
		return volumeCreateCommand{o.nodes, volname}
	case 4:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		initialCluster := make([]string, len(o.nodes))
		for i, n := range o.nodes {
			initialCluster[i] = n.Address + "=https://" + n.Address + ":2380"
		}
		paramsMap := make(map[string]ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = etcdBuiltInParams(n, initialCluster, "new")
		}
		return runContainerCommand{
			nodes:     o.nodes,
			name:      etcdContainerName,
			img:       EtcdImage,
			opts:      opts,
			paramsMap: paramsMap,
			extra:     o.params.ServiceParams,
		}
	case 5:
		o.step++
		return waitEtcdSyncCommand{o.endpoints, false}
	case 6:
		o.step++
		return setupEtcdAuthCommand{o.endpoints}
	default:
		return nil
	}
}

type etcdStartOp struct {
	nodes     []*Node
	params    EtcdParams
	step      int
	makeFiles *makeFilesCommand
}

// EtcdStartOp returns an Operator to start etcd containers.
func EtcdStartOp(nodes []*Node, params EtcdParams) Operator {
	return &etcdStartOp{
		nodes:     nodes,
		params:    params,
		makeFiles: &makeFilesCommand{nodes: nodes},
	}
}

func (o *etcdStartOp) Name() string {
	return "etcd-start"
}

func (o *etcdStartOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return prepareEtcdCertificatesCommand{o.makeFiles}
	case 1:
		o.step++
		return o.makeFiles
	case 2:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + etcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		paramsMap := make(map[string]ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = etcdBuiltInParams(n, nil, "")
		}
		return runContainerCommand{
			nodes:     o.nodes,
			name:      etcdContainerName,
			img:       EtcdImage,
			opts:      opts,
			paramsMap: paramsMap,
			extra:     o.params.ServiceParams,
		}
	case 3:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.nodes), false}
	default:
		return nil
	}
}

type etcdAddMemberOp struct {
	endpoints  []string
	targetNode *Node
	params     EtcdParams
	step       int
	makeFiles  *makeFilesCommand
}

// EtcdAddMemberOp returns an Operator to add member to etcd cluster.
func EtcdAddMemberOp(cp []*Node, targetNode *Node, params EtcdParams) Operator {
	return &etcdAddMemberOp{
		endpoints:  etcdEndpoints(cp),
		targetNode: targetNode,
		params:     params,
		makeFiles:  &makeFilesCommand{nodes: []*Node{targetNode}},
	}
}

func (o *etcdAddMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *etcdAddMemberOp) NextCommand() Commander {
	volname := etcdVolumeName(o.params)
	extra := o.params.ServiceParams

	nodes := []*Node{o.targetNode}
	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{nodes, EtcdImage}
	case 1:
		o.step++
		return stopContainerCommand{o.targetNode, etcdContainerName}
	case 2:
		o.step++
		return volumeRemoveCommand{nodes, volname}
	case 3:
		o.step++
		return volumeCreateCommand{nodes, volname}
	case 4:
		o.step++
		return prepareEtcdCertificatesCommand{o.makeFiles}
	case 5:
		o.step++
		return o.makeFiles
	case 6:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		return addEtcdMemberCommand{o.endpoints, o.targetNode, opts, extra}
	case 7:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints([]*Node{o.targetNode}), false}
	}
	return nil
}

type addEtcdMemberCommand struct {
	endpoints []string
	node      *Node
	opts      []string
	extra     ServiceParams
}

func (c addEtcdMemberCommand) Run(ctx context.Context, inf Infrastructure) error {
	cli, err := inf.NewEtcdClient(c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return err
	}
	members := resp.Members

	inMember := false
	for _, m := range members {
		inMember, err = addressInURLs(c.node.Address, m.PeerURLs)
		if err != nil {
			return err
		}
		if inMember {
			break
		}
	}

	if !inMember {
		ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
		defer cancel()
		resp, err := cli.MemberAdd(ct, []string{fmt.Sprintf("https://%s:2380", c.node.Address)})
		if err != nil {
			return err
		}
		members = resp.Members
	}
	// gofail: var etcdAfterMemberAdd struct{}
	ce := Docker(inf.Agent(c.node.Address))
	ss, err := ce.Inspect([]string{etcdContainerName})
	if err != nil {
		return err
	}
	if ss[etcdContainerName].Running {
		return nil
	}

	var initialCluster []string
	for _, m := range members {
		for _, u := range m.PeerURLs {
			if len(m.Name) == 0 {
				initialCluster = append(initialCluster, c.node.Address+"="+u)
			} else {
				initialCluster = append(initialCluster, m.Name+"="+u)
			}
		}
	}

	return ce.RunSystem(etcdContainerName, EtcdImage, c.opts, etcdBuiltInParams(c.node, initialCluster, "existing"), c.extra)
}

func (c addEtcdMemberCommand) Command() Command {
	return Command{
		Name:   "add-etcd-member",
		Target: c.node.Address,
	}
}

type waitEtcdSyncCommand struct {
	endpoints       []string
	checkRedundancy bool
}

func (c waitEtcdSyncCommand) try(ctx context.Context, inf Infrastructure) error {
	cli, err := inf.NewEtcdClient(c.endpoints)
	if err != nil {
		return err
	}

	ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
	defer cancel()
	resp, err := cli.Grant(ct, 10)
	if err != nil {
		return err
	}
	if resp.ID == clientv3.NoLease {
		return errors.New("no lease")
	}

	if !c.checkRedundancy {
		return nil
	}

	healthyMemberCount := 0
	for _, ep := range c.endpoints {
		ct2, cancel2 := context.WithTimeout(ctx, defaultEtcdTimeout)
		_, err = cli.Status(ct2, ep)
		cancel2()
		if err == nil {
			healthyMemberCount++
		}
	}
	if healthyMemberCount <= int(len(c.endpoints)+1)/2 {
		return errors.New("etcd cluster is not redundant enough")
	}
	return nil
}

func (c waitEtcdSyncCommand) Run(ctx context.Context, inf Infrastructure) error {
	for i := 0; i < 9; i++ {
		err := c.try(ctx, inf)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	// last try
	return c.try(ctx, inf)
}

func (c waitEtcdSyncCommand) Command() Command {
	return Command{
		Name:   "wait-etcd-sync",
		Target: strings.Join(c.endpoints, ","),
	}
}

type setupEtcdAuthCommand struct {
	endpoints []string
}

func (c setupEtcdAuthCommand) Run(ctx context.Context, inf Infrastructure) error {
	cli, err := inf.NewEtcdClient(c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	err = AddUserRole(ctx, cli, "root", "")
	if err != nil {
		return err
	}
	_, err = cli.UserGrantRole(ctx, "root", "root")
	if err != nil {
		return err
	}

	err = AddUserRole(ctx, cli, "kube-apiserver", "/registry/")
	if err != nil {
		return err
	}

	_, err = cli.AuthEnable(ctx)
	return err
}

func (c setupEtcdAuthCommand) Command() Command {
	return Command{
		Name:   "setup-etcd-auth",
		Target: strings.Join(c.endpoints, ","),
	}
}

// AddUserRole create etcd user and role
func AddUserRole(ctx context.Context, cli *clientv3.Client, name, prefix string) error {
	r := make([]byte, 32)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}

	_, err = cli.UserAdd(ctx, name, hex.EncodeToString(r))
	if err != nil {
		return err
	}

	if prefix == "" {
		return nil
	}

	_, err = cli.RoleAdd(ctx, name)
	if err != nil {
		return err
	}

	_, err = cli.RoleGrantPermission(ctx, name, prefix, clientv3.GetPrefixRangeEnd(prefix), clientv3.PermissionType(clientv3.PermReadWrite))
	if err != nil {
		return err
	}

	_, err = cli.UserGrantRole(ctx, name, name)
	if err != nil {
		return err
	}

	return nil
}

// GetUserRoles get roles of target user
func GetUserRoles(ctx context.Context, cli *clientv3.Client, user string) ([]string, error) {
	resp, err := cli.UserGet(ctx, user)
	return resp.Roles, err
}

type removeEtcdMemberCommand struct {
	endpoints []string
	ids       []uint64
}

func (c removeEtcdMemberCommand) Run(ctx context.Context, inf Infrastructure) error {
	cli, err := inf.NewEtcdClient(c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, id := range c.ids {
		ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
		_, err := cli.MemberRemove(ct, id)
		cancel()
		if err != nil {
			return err
		}
	}
	// gofail: var etcdAfterMemberRemove struct{}
	return nil
}

func (c removeEtcdMemberCommand) Command() Command {
	idStrs := make([]string, len(c.ids))
	for i, id := range c.ids {
		idStrs[i] = strconv.FormatUint(id, 10)
	}
	return Command{
		Name:   "remove-etcd-member",
		Target: strings.Join(idStrs, ","),
	}
}

// EtcdWaitClusterOp returns an Operator to wait until etcd cluster becomes healthy
func EtcdWaitClusterOp(nodes []*Node) Operator {
	return &etcdWaitClusterOp{
		endpoints: etcdEndpoints(nodes),
	}
}

type etcdWaitClusterOp struct {
	endpoints []string
	executed  bool
}

func (o *etcdWaitClusterOp) Name() string {
	return "etcd-wait-cluster"
}

func (o *etcdWaitClusterOp) NextCommand() Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return waitEtcdSyncCommand{o.endpoints, false}
}

type etcdRemoveMemberOp struct {
	endpoints []string
	ids       []uint64
	executed  bool
}

// EtcdRemoveMemberOp returns an Operator to remove member from etcd cluster.
func EtcdRemoveMemberOp(cp []*Node, ids []uint64) Operator {
	return &etcdRemoveMemberOp{
		endpoints: etcdEndpoints(cp),
		ids:       ids,
	}
}

func (o *etcdRemoveMemberOp) Name() string {
	return "etcd-remove-member"
}

func (o *etcdRemoveMemberOp) NextCommand() Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return removeEtcdMemberCommand{o.endpoints, o.ids}
}

type etcdDestroyMemberOp struct {
	endpoints []string
	targets   []*Node
	ids       []uint64
	params    EtcdParams
	step      int
}

// EtcdDestroyMemberOp create new etcdDestroyMemberOp instance
func EtcdDestroyMemberOp(cp []*Node, targets []*Node, ids []uint64) Operator {
	return &etcdDestroyMemberOp{
		endpoints: etcdEndpoints(cp),
		targets:   targets,
		ids:       ids,
	}
}

func (o *etcdDestroyMemberOp) Name() string {
	return "etcd-destroy-member"
}

func (o *etcdDestroyMemberOp) NextCommand() Commander {
	// Destroy need to remove etcd data first.
	// Otherwise, if CKE crashes just after removing a member from cluster
	// but leave the data, next time CKE just starts the container without
	// adding it as a new member.

	switch o.step {
	case 0:
		o.step++
		return killContainersCommand{o.targets, etcdContainerName}
	case 1:
		o.step++
		return volumeRemoveCommand{o.targets, etcdVolumeName(o.params)}
	case 2:
		o.step++
		return removeEtcdMemberCommand{o.endpoints, o.ids}
	case 3:
		o.step++
		return waitEtcdSyncCommand{o.endpoints, false}
	}
	return nil
}

// EtcdRestartOp create new etcdRestartOp instance
func EtcdRestartOp(cpNodes []*Node, target *Node, params EtcdParams) Operator {
	return &etcdRestartOp{
		cpNodes: cpNodes,
		target:  target,
		params:  params,
	}
}

type etcdRestartOp struct {
	cpNodes []*Node
	target  *Node
	params  EtcdParams
	step    int
}

func (o *etcdRestartOp) Name() string {
	return "etcd-restart"
}

func (o *etcdRestartOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.cpNodes), true}
	case 1:
		o.step++
		return imagePullCommand{[]*Node{o.target}, EtcdImage}
	case 2:
		o.step++
		return stopContainerCommand{o.target, etcdContainerName}
	case 3:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + etcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		var initialCluster []string
		for _, n := range o.cpNodes {
			initialCluster = append(initialCluster, n.Address+"=https://"+n.Address+":2380")
		}
		return runContainerCommand{
			nodes:  []*Node{o.target},
			name:   etcdContainerName,
			img:    EtcdImage,
			opts:   opts,
			params: etcdBuiltInParams(o.target, initialCluster, "new"),
			extra:  o.params.ServiceParams,
		}
	}
	return nil
}
