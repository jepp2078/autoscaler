/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rancher

import (
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"fmt"
)

type nodePoolCache struct {
	registeredNodePools     []cloudprovider.NodeGroup
	instanceToNodePool      map[RancherRef]cloudprovider.NodeGroup
	notInRegisteredNodePool map[RancherRef]bool
	mutex              sync.Mutex
	interrupt          chan struct{}
}

func newNodePoolCache() (*nodePoolCache, error) {
	cache := &nodePoolCache{
		registeredNodePools:     make([]cloudprovider.NodeGroup, 0),
		instanceToNodePool:      make(map[RancherRef]cloudprovider.NodeGroup),
		notInRegisteredNodePool: make(map[RancherRef]bool),
		interrupt:          make(chan struct{}),
	}

	go wait.Until(func() {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()
		if err := cache.regenerate(); err != nil {
			glog.Errorf("Error while regenerating NodePool cache: %v", err)
		}
	}, time.Hour, cache.interrupt)

	return cache, nil
}

// Register registers a node group if it hasn't been registered.
func (m *nodePoolCache) Register(nodePool cloudprovider.NodeGroup) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i := range m.registeredNodePools {
		if existing := m.registeredNodePools[i]; existing.Id() == nodePool.Id() {
			if reflect.DeepEqual(existing, nodePool) {
				return false
			}

			m.registeredNodePools[i] = nodePool
			glog.V(4).Infof("NodePool %q updated", nodePool.Id())
			m.invalidateUnownedInstanceCache()
			return true
		}
	}

	glog.V(4).Infof("Registering NodePool %q", nodePool.Id())
	m.registeredNodePools = append(m.registeredNodePools, nodePool)
	m.invalidateUnownedInstanceCache()
	return true
}

func (m *nodePoolCache) invalidateUnownedInstanceCache() {
	glog.V(4).Info("Invalidating unowned instance cache")
	m.notInRegisteredNodePool = make(map[RancherRef]bool)
}

// Unregister NodePool. Returns true if the NodePool was unregistered.
func (m *nodePoolCache) Unregister(nodePool cloudprovider.NodeGroup) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	updated := make([]cloudprovider.NodeGroup, 0, len(m.registeredNodePools))
	changed := false
	for _, existing := range m.registeredNodePools {
		if existing.Id() == nodePool.Id() {
			glog.V(1).Infof("Unregistered NodePool %s", nodePool.Id())
			changed = true
			continue
		}
		updated = append(updated, existing)
	}
	m.registeredNodePools = updated
	return changed
}

func (m *nodePoolCache) get() []cloudprovider.NodeGroup {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.registeredNodePools
}

// FindForInstance returns NodePool of the given Instance
func (m *nodePoolCache) FindForInstance(instance *RancherRef) (cloudprovider.NodeGroup, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.notInRegisteredNodePool[*instance] {
		// We already know we don't own this instance. Return early and avoid
		// additional calls.
		return nil, nil
	}

	if nodePool, found := m.instanceToNodePool[*instance]; found {
		return nodePool, nil
	}

	if err := m.regenerate(); err != nil {
		return nil, fmt.Errorf("Error while looking for NodePool for instance %+v, error: %v", *instance, err)
	}
	if config, found := m.instanceToNodePool[*instance]; found {
		return config, nil
	}

	m.notInRegisteredNodePool[*instance] = true
	return nil, nil
}

// Cleanup closes the channel to signal the go routine to stop that is handling the cache
func (m *nodePoolCache) Cleanup() {
	close(m.interrupt)
}

func (m *nodePoolCache) regenerate() error {
	newCache := make(map[RancherRef]cloudprovider.NodeGroup)

	for _, nsg := range m.registeredNodePools {
		instances, err := nsg.Nodes()
		if err != nil {
			return err
		}

		for _, instance := range instances {
			ref := RancherRef{Name: instance}
			newCache[ref] = nsg
		}
	}

	m.instanceToNodePool = newCache
	return nil
}
