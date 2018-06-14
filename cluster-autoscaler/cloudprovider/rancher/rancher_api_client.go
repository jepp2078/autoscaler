package rancher

import (
	"net/http"
	"fmt"
	"encoding/json"
	"bytes"
)

type baseAPIClient struct {
	ClusterID         string
	RancherToken      string
	RancherURI        string
}

func NewBaseAPIClient(cfg *Config) *baseAPIClient{
	return &baseAPIClient{
		ClusterID: cfg.ClusterID,
		RancherToken: cfg.RancherToken,
		RancherURI: cfg.RancherURI,
	}
}

type rancherNodePoolAPIClient struct {
	client    baseAPIClient
	resource  string
}

func NewRancherNodePoolAPIClient(cfg *Config) *rancherNodePoolAPIClient{
	return &rancherNodePoolAPIClient{client: *NewBaseAPIClient(cfg), resource: "nodePools/"}
}

// NodePoolAPIClient defines needed functions for the rancher node-pool api.
type NodePoolAPIClient interface {
	UpdateNodePoolCapacity(nodePoolName string, size int64) (resp *http.Response, err error)
	GetNodePool(nodePoolName string) (result *RancherNodePool, err error)
}

func (nc *rancherNodePoolAPIClient) GetNodePool(nodePoolName string) (result *RancherNodePool, err error){

	// Build URL
	url := fmt.Sprintf("%s%s%s", nc.client.RancherURI, nc.resource, nodePoolName)

	// Build the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Set("Authorization", nc.client.RancherToken)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err)
	}

	defer resp.Body.Close()

	var nodePool *RancherNodePool

	if err := json.NewDecoder(resp.Body).Decode(&nodePool); err != nil {
		return nil, fmt.Errorf("error parsing json: %s", err)
	}

	return nodePool, nil
}

func (nc *rancherNodePoolAPIClient) UpdateNodePoolCapacity(nodePoolName string, size int64) (err error){
	// Build URL
	url := fmt.Sprintf("%s%s%s", nc.client.RancherURI, nc.resource, nodePoolName)

	currNodePool, err := nc.GetNodePool(nodePoolName)
	currNodePool.Quantity = int(size)

	currNodePoolJson, err := json.Marshal(currNodePool)
	if err != nil {
		return fmt.Errorf("error marshalling json: %s", err)
	}

	// Build the request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(currNodePoolJson))
	if err != nil {
		return fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Set("Authorization", nc.client.RancherToken)
	client := &http.Client{}

	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}

	return nil
}

type rancherNodeAPIClient struct {
	client    baseAPIClient
	resource  string
}

func NewRancherNodeAPIClient(cfg *Config) *rancherNodeAPIClient{
	return &rancherNodeAPIClient{client: *NewBaseAPIClient(cfg), resource: "nodes/"}
}

// NodeAPIClient defines needed functions for the rancher node api.
type NodeAPIClient interface {
	DeleteNode(nodeName string) (resp *http.Response, err error)
	ListNodes(nodePoolName string) (result []*RancherNode, err error)
}

//TODO(Jepp2078): Change this func to a single call when rancher adds node.Spec.ProviderID
func (nc *rancherNodeAPIClient) DeleteNode(nodeName string) (err error){
	node, err := nc.GetNodeByName(nodeName)
	if err != nil {
		return err
	}

	// Build URL
	url := fmt.Sprintf("%s%s%s", nc.client.RancherURI, nc.resource, node.ID)

	// Build the request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Set("Authorization", nc.client.RancherToken)
	client := &http.Client{}

	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}

	return nil
}

func (nc *rancherNodeAPIClient) ListNodes(nodePoolName string) (result []*RancherNode, err error){
	// Build URL
	url := fmt.Sprintf("%s%s?nodePoolId=%s", nc.client.RancherURI, nc.resource, nodePoolName)

	// Build the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Set("Authorization", nc.client.RancherToken)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err)
	}

	defer resp.Body.Close()

	var nodePool *RancherNodeContainer

	if err := json.NewDecoder(resp.Body).Decode(&nodePool); err != nil {
		return nil, fmt.Errorf("error parsing json: %s", err)
	}

	return nodePool.Data, nil
}

//TODO(Jepp2078): This can be removed When rancher addes noce.Spec.ProviderID
func (nc *rancherNodeAPIClient) GetNodeByName(nodeName string) (result *RancherNode, err error){
	// Build URL
	url := fmt.Sprintf("%s/cluster/%s/%s?name=%s", nc.client.RancherURI, nc.client.ClusterID, nc.resource, nodeName)

	// Build the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Set("Authorization", nc.client.RancherToken)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err)
	}

	defer resp.Body.Close()

	var node *RancherNodeContainer

	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("error parsing json: %s", err)
	}

	if len(node.Data) > 1{
		return nil, fmt.Errorf("error getting node: %s is ambiguous", nodeName)
	}

	return node.Data[0], nil
}