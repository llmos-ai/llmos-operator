package template

type CephConfig struct {
	// name is the prefix name of the ceph cluster
	name string
	// namespace is the namespace of the ceph cluster
	namespace string
	// operatorNamespace is the rook-operator namespace
	operatorNamespace string
	BlockPoolConfig   BlockPoolConfig
	FilesystemConfig  FilesystemConfig
}

type BlockPoolConfig struct {
	Name        string `default:"ceph-blockpool"`
	Replicas    int    `default:"2"`
	MinReplicas int    `default:"1"`
}

type FilesystemConfig struct {
	Name                 string `default:"ceph-filesystem"`
	DataPoolReplicas     int    `default:"2"`
	MetadataPoolReplicas int    `default:"1"`
}

func NewCephConfig(name, namespace, operatorNs string) *CephConfig {
	return &CephConfig{
		name:              name,
		namespace:         namespace,
		operatorNamespace: operatorNs,
		BlockPoolConfig: BlockPoolConfig{
			Name:        "ceph-blockpool",
			Replicas:    2,
			MinReplicas: 1,
		},
		FilesystemConfig: FilesystemConfig{
			Name:                 "ceph-filesystem",
			DataPoolReplicas:     2,
			MetadataPoolReplicas: 1,
		},
	}
}

func (c *CephConfig) Name() string {
	return c.name
}

func (c *CephConfig) Namespace() string {
	return c.namespace
}

func (c *CephConfig) OperatorNamespace() string {
	return c.operatorNamespace
}

func (c *CephConfig) BlockPoolName() string {
	return c.BlockPoolConfig.Name
}

func (c *CephConfig) BlockPoolReplicas() int {
	return c.BlockPoolConfig.Replicas
}

func (c *CephConfig) BlockPoolMinReplicas() int {
	return c.BlockPoolConfig.MinReplicas
}

func (c *CephConfig) FilesystemName() string {
	return c.FilesystemConfig.Name
}

func (c *CephConfig) FilesystemPoolName() string {
	return c.FilesystemConfig.Name + "-replicated"
}

func (c *CephConfig) FilesystemDataPoolReplicas() int {
	return c.FilesystemConfig.DataPoolReplicas
}

func (c *CephConfig) FilesystemMetadataPoolReplicas() int {
	return c.FilesystemConfig.MetadataPoolReplicas
}

func (c *CephConfig) GetToolboxName() string {
	return c.name + "-tools"
}
