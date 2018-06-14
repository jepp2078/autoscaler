package rancher

func BuildRancherClient(cfg *Config) (*rancherClient, error){
	rancherNodeClient := NewRancherNodeClient(cfg)
	rancherNodePoolClient := NewRancherNodePoolClient(cfg)

	client := &rancherClient{
		nodeClient:         rancherNodeClient,
		nodePoolClient:     rancherNodePoolClient,
	}

	return client, nil
}

type rancherNodeClient struct {
	client    rancherNodeAPIClient
	resource  string
}

func NewRancherNodeClient(cfg *Config) *rancherNodeClient{
	return &rancherNodeClient{client: *NewRancherNodeAPIClient(cfg)}
}

// NodeClient defines needed functions for rancher Nodes.
type NodeClient interface {
	Delete(nodeName string) (err error)
	List(nodePoolName string) (result []*RancherNode, err error)
}

func (nc *rancherNodeClient) Delete(nodeName string) (err error){
	err = nc.client.DeleteNode(nodeName)
	if err != nil {
		return err
	}
	return nil
}

func (nc *rancherNodeClient) List(nodePoolName string) (result []*RancherNode, err error){
	result, err = nc.client.ListNodes(nodePoolName)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type rancherNodePoolClient struct {
	client    rancherNodePoolAPIClient
}

func NewRancherNodePoolClient(cfg *Config) *rancherNodePoolClient{
	return &rancherNodePoolClient{client: *NewRancherNodePoolAPIClient(cfg)}
}

// NodeClient defines needed functions for rancher Nodes.
type NodePoolClient interface {
	SetDesiredCapacity(nodePoolName string, size int64) (err error)
	Get(nodePoolName string) (result *RancherNodePool, err error)
}

func (nc *rancherNodePoolClient) SetDesiredCapacity(nodePoolName string, size int64) (err error){
	err = nc.client.UpdateNodePoolCapacity(nodePoolName, size)
	if err != nil {
		return err
	}
	return nil
}

func (nc *rancherNodePoolClient) Get(nodePoolName string) (result *RancherNodePool, err error){
	result, err = nc.client.GetNodePool(nodePoolName)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type rancherClient struct {
	nodeClient     NodeClient
	nodePoolClient NodePoolClient
}