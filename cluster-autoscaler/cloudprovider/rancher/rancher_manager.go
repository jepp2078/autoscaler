package rancher

import (
	"time"
	"io"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"gopkg.in/gcfg.v1"
	"github.com/golang/glog"
	"os"
	"fmt"
	"k8s.io/autoscaler/cluster-autoscaler/config/dynamic"
)

const (
	scaleToZeroSupportedStandard = false
	refreshInterval         = 1 * time.Minute
)

// Config holds the configuration parsed from the --cloud-config flag
type Config struct {
	ClusterID          string `json:"clusterId" yaml:"clusterId"`
	RancherToken       string `json:"rancherToken" yaml:"rancherToken"`
	RancherURI         string `json:"rancherUri" yaml:"rancherUri"`
}

// RancherManager is handles rancher communication
type RancherManager struct {
	service              *rancherClient
	nodePoolCache        *nodePoolCache
	lastRefresh          time.Time
	explicitlyConfigured map[string]bool
}

func BuildRancherManager(configReader io.Reader, discoveryOpts cloudprovider.NodeGroupDiscoveryOptions) (*RancherManager, error) {
	var cfg Config

	if configReader != nil {
		if err := gcfg.ReadInto(&cfg, configReader); err != nil {
			glog.Errorf("Couldn't read config: %v", err)
			return nil, err
		}
	} else {
		cfg.ClusterID = os.Getenv("CLUSTER_ID")
		cfg.RancherToken = os.Getenv("RANCHER_TOKEN")
		cfg.RancherURI = os.Getenv("RANCHER_URI")
	}

	service, err := BuildRancherClient(&cfg)
	if err != nil {
		return nil, err
	}

	manager := &RancherManager{
		service:      service,
		explicitlyConfigured: make(map[string]bool),
	}

	cache, err := newNodePoolCache()
	if err != nil {
		return nil, err
	}
	manager.nodePoolCache = cache

	if err := manager.fetchExplicitNodePools(discoveryOpts.NodeGroupSpecs); err != nil {
		return nil, err
	}

	if err := manager.forceRefresh(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *RancherManager) fetchExplicitNodePools(specs []string) error {
	changed := false
	glog.V(4).Infof("fetchExplicitNodePools %s", specs)
	for _, spec := range specs {
		nodePool, err := m.buildNodePoolFromSpec(spec)
		if err != nil {
			return fmt.Errorf("failed to parse node group spec: %v", err)
		}
		if m.RegisterNodePool(nodePool) {
			changed = true
		}
		m.explicitlyConfigured[nodePool.Id()] = true
	}

	if changed {
		if err := m.regenerateCache(); err != nil {
			return err
		}
	}
	return nil
}

func (m *RancherManager) buildNodePoolFromSpec(spec string) (cloudprovider.NodeGroup, error) {
	scaleToZeroSupported := scaleToZeroSupportedStandard

	s, err := dynamic.SpecFromString(spec, scaleToZeroSupported)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node group spec: %v", err)
	}

	return NewNodePool(s, m)
}

// GetNodePoolSize gets NodePool size.
func (m *RancherManager) GetNodePoolSize(nodePool *NodePool) (int64, error) {
	nodes, err := m.service.nodeClient.List(nodePool.Name)
	if err != nil {
		return -1, err
	}
	result := int64(len(nodes))
	return result, nil
}

// SetNodePoolSize sets NodePool size.
func (m *RancherManager) SetNodePoolSize(nodePool *NodePool, size int64) error {
	glog.V(0).Infof("Setting NodePool %s size to %d", nodePool.Id(), size)
	err := m.service.nodePoolClient.SetDesiredCapacity(nodePool.Name, size)
	if err != nil {
		return err
	}
	return nil
}

// GetNodePoolNodes returns NodePool nodes.
func (m *RancherManager) GetNodePoolNodes(nodePool *NodePool) ([]string, error) {
	result := make([]string, 0)
	nodes, err := m.service.nodeClient.List(nodePool.Name)
	if err != nil {
		return []string{}, err
	}
	for _, instance := range nodes {
		result = append(result,
			fmt.Sprintf("%s", instance.NodeName))
	}
	return result, nil
}

// DeleteInstances deletes the given instances. All instances must be controlled by the same NodePool.
func (m *RancherManager) DeleteInstances(instances []*RancherRef) error {
	if len(instances) == 0 {
		return nil
	}

	for _, instance := range instances {
		err := m.service.nodeClient.Delete(instance.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
func (m *RancherManager) Refresh() error {
	if m.lastRefresh.Add(refreshInterval).After(time.Now()) {
		return nil
	}
	return m.forceRefresh()
}

func (m *RancherManager) forceRefresh() error {
	//TODO(Jepp2078): If and when Rancher allows us to tag node-pools, enable auto-discovery of node-pools.
	m.lastRefresh = time.Now()
	glog.V(2).Infof("Refreshed NodePool list, next refresh after %v", m.lastRefresh.Add(refreshInterval))
	return nil
}

func (m *RancherManager) getNodePools() []cloudprovider.NodeGroup {
	return m.nodePoolCache.get()
}

// GetNodePoolForInstance returns NodePoolConfig of the given Instance
func (m *RancherManager) GetNodePoolForInstance(instance *RancherRef) (cloudprovider.NodeGroup, error) {
	return m.nodePoolCache.FindForInstance(instance)
}

// RegisterNodePool registers an NodePool.
func (m *RancherManager) RegisterNodePool(nodePool cloudprovider.NodeGroup) bool {
	return m.nodePoolCache.Register(nodePool)
}

// UnregisterNodePool unregisters an NodePool.
func (m *RancherManager) UnregisterNodePool(nodePool cloudprovider.NodeGroup) bool {
	return m.nodePoolCache.Unregister(nodePool)
}


func (m *RancherManager) regenerateCache() error {
	m.nodePoolCache.mutex.Lock()
	defer m.nodePoolCache.mutex.Unlock()
	return m.nodePoolCache.regenerate()
}

// Cleanup the NodePool cache.
func (m *RancherManager) Cleanup() {
	m.nodePoolCache.Cleanup()
}