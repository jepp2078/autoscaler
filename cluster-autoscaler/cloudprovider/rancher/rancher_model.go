package rancher

import "time"

type RancherNodePool struct {
	Actions struct {
	} `json:"actions"`
	Annotations struct {
		LifecycleCattleIoCreateNodepoolProvisioner string `json:"lifecycle.cattle.io/create.nodepool-provisioner"`
	} `json:"annotations"`
	BaseType       string    `json:"baseType"`
	ClusterID      string    `json:"clusterId"`
	ControlPlane   bool      `json:"controlPlane"`
	Created        time.Time `json:"created"`
	CreatedTS      int64     `json:"createdTS"`
	CreatorID      string    `json:"creatorId"`
	DisplayName    string    `json:"displayName"`
	Etcd           bool      `json:"etcd"`
	HostnamePrefix string    `json:"hostnamePrefix"`
	ID             string    `json:"id"`
	Links          struct {
		Nodes  string `json:"nodes"`
		Remove string `json:"remove"`
		Self   string `json:"self"`
		Update string `json:"update"`
	} `json:"links"`
	Name            string      `json:"name"`
	NamespaceID     interface{} `json:"namespaceId"`
	NodeAnnotations interface{} `json:"nodeAnnotations"`
	NodeLabels      interface{} `json:"nodeLabels"`
	NodeTemplateID  string      `json:"nodeTemplateId"`
	Quantity        int         `json:"quantity"`
	State           string      `json:"state"`
	Status          struct {
		Conditions []struct {
			LastUpdateTime time.Time `json:"lastUpdateTime"`
			Status         string    `json:"status"`
			Type           string    `json:"type"`
		} `json:"conditions"`
		Type string `json:"type"`
	} `json:"status"`
	Transitioning        string `json:"transitioning"`
	TransitioningMessage string `json:"transitioningMessage"`
	Type                 string `json:"type"`
	UUID                 string `json:"uuid"`
	Worker               bool   `json:"worker"`
	Nodes		   []*RancherNode
}

type RancherNode struct {
	ClusterID         string       `json:"clusterId"`
	Hostname          string       `json:"hostname"`
	ID                string       `json:"id"`
	NodeName          string       `json:"nodeName"`
	NodePoolID        string       `json:"nodePoolId"`
	UUID              string       `json:"uuid"`
}

type RancherNodeContainer struct {
	Data        []*RancherNode         `json:"data"`
}