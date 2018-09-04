package placemat

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// Cluster is a set of resources in a virtual data center.
type Cluster struct {
	Networks    []*Network
	Images      []*Image
	DataFolders []*DataFolder
	Nodes       []*Node
	Pods        []*Pod

	// private fields will be initialized by Resolve.
	netMap    map[string]*Network
	imageMap  map[string]*Image
	folderMap map[string]*DataFolder
	nodeMap   map[string]*Node
	podMap    map[string]*Pod
}

// Append appends another cluster into the receiver.
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Images = append(c.Images, other.Images...)
	c.DataFolders = append(c.DataFolders, other.DataFolders...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.Pods = append(c.Pods, other.Pods...)
	return c
}

// Resolve resolves inter-resource references and checks unique constraints.
func (c *Cluster) Resolve() error {
	c.netMap = make(map[string]*Network)
	for _, n := range c.Networks {
		if _, ok := c.netMap[n.Name]; ok {
			return errors.New("duplicate network: " + n.Name)
		}
		c.netMap[n.Name] = n
	}

	c.imageMap = make(map[string]*Image)
	for _, i := range c.Images {
		if _, ok := c.imageMap[i.Name]; ok {
			return errors.New("duplicate image: " + i.Name)
		}
		c.imageMap[i.Name] = i
	}

	c.folderMap = make(map[string]*DataFolder)
	for _, f := range c.DataFolders {
		if _, ok := c.folderMap[f.Name]; ok {
			return errors.New("duplicate data folder: " + f.Name)
		}
		c.folderMap[f.Name] = f
	}

	c.nodeMap = make(map[string]*Node)
	for _, n := range c.Nodes {
		err := n.Resolve(c)
		if err != nil {
			return err
		}
		if _, ok := c.nodeMap[n.Name]; ok {
			return errors.New("duplicate node: " + n.Name)
		}
		c.nodeMap[n.Name] = n
	}

	c.podMap = make(map[string]*Pod)
	for _, p := range c.Pods {
		err := p.Resolve(c)
		if err != nil {
			return err
		}
		if _, ok := c.podMap[p.Name]; ok {
			return errors.New("duplicate pod: " + p.Name)
		}
		c.podMap[p.Name] = p
	}

	return nil
}

// Cleanup remaining resources
func (c *Cluster) Cleanup(r *Runtime) error {
	err := CleanupNodes(r, c.Nodes)
	if err != nil {
		return err
	}

	CleanupNetworks(r, c)

	return CleanupRootfs()
}

// GetNetwork looks up the network by name.
// It returns non-nil error if the named network is not found.
func (c *Cluster) GetNetwork(name string) (*Network, error) {
	n, ok := c.netMap[name]
	if !ok {
		return nil, errors.New("no such network: " + name)
	}
	return n, nil
}

// GetImage looks up the image by name.
// It returns non-nil error if the named image is not found.
func (c *Cluster) GetImage(name string) (*Image, error) {
	i, ok := c.imageMap[name]
	if !ok {
		return nil, errors.New("no such image: " + name)
	}
	return i, nil
}

// GetDataFolder looks up the data folder by name.
// It returns non-nil error if the named folder is not found.
func (c *Cluster) GetDataFolder(name string) (*DataFolder, error) {
	f, ok := c.folderMap[name]
	if !ok {
		return nil, errors.New("no such data folder: " + name)
	}
	return f, nil
}

// GetNode looks up the node by name.
// It returns non-nil error if the named node is not found.
func (c *Cluster) GetNode(name string) (*Node, error) {
	n, ok := c.nodeMap[name]
	if !ok {
		return nil, errors.New("no such node: " + name)
	}
	return n, nil
}

// GetPod looks up the pod by name.
// It returns non-nil error if the named pod is not found.
func (c *Cluster) GetPod(name string) (*Pod, error) {
	p, ok := c.podMap[name]
	if !ok {
		return nil, errors.New("no such pod: " + name)
	}
	return p, nil
}

// Start constructs the virtual data center with given resources.
// It stop when ctx is cancelled.
func (c *Cluster) Start(ctx context.Context, r *Runtime) error {
	defer os.RemoveAll(r.tempDir)

	if r.force {
		err := c.Cleanup(r)
		if err != nil {
			return err
		}
	}

	root, err := NewRootfs()
	if err != nil {
		return err
	}
	defer root.Destroy()

	err = createNatRules()
	if err != nil {
		return err
	}
	defer destroyNatRules()

	for _, n := range c.Networks {
		log.Info("Creating network", map[string]interface{}{"name": n.Name})
		err := n.Create(r.nameGenerator())
		if err != nil {
			return err
		}
		defer n.Destroy()
	}

	for _, df := range c.DataFolders {
		log.Info("initializing data folder", map[string]interface{}{
			"name": df.Name,
		})
		err := df.Prepare(ctx, r.tempDir, r.dataCache)
		if err != nil {
			return err
		}
	}

	for _, img := range c.Images {
		log.Info("initializing image resource", map[string]interface{}{
			"name": img.Name,
		})
		err := img.Prepare(ctx, r.imageCache)
		if err != nil {
			return err
		}
	}

	for _, p := range c.Pods {
		err := p.Prepare(ctx)
		if err != nil {
			return err
		}
	}

	nodeCh := make(chan bmcInfo, len(c.Nodes))

	var mu sync.Mutex
	vms := make(map[string]*NodeVM)

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.Nodes {
		n := n
		env.Go(func(ctx2 context.Context) error {
			// reference the original context because ctx2 will soon be cancelled.
			vm, err := n.Start(ctx, r, nodeCh)
			if err != nil {
				return err
			}
			mu.Lock()
			vms[n.SMBIOS.Serial] = vm
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err = env.Wait()
	defer func() {
		for _, vm := range vms {
			vm.cleanup()
		}
	}()
	if err != nil {
		return err
	}

	bmcServer := newBMCServer(vms, c.Networks, nodeCh)

	env = cmd.NewEnvironment(ctx)
	env.Go(bmcServer.handleNode)
	for _, p := range c.Pods {
		p := p
		env.Go(func(ctx context.Context) error {
			return p.Start(ctx, r, root.Path())
		})
	}
	for _, vm := range vms {
		vm := vm
		env.Go(func(ctx context.Context) error {
			return vm.cmd.Wait()
		})
	}
	env.Stop()

	return env.Wait()
}
