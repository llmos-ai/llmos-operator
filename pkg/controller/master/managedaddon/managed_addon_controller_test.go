package managedaddon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestMergeValuesContent(t *testing.T) {
	type input struct {
		key           string
		defaultValues string
		valueContent  string
	}
	type output struct {
		values string
		err    error
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "merge values",
			given: input{
				key: "test",
				defaultValues: `
operatorNamespace: storage-system
clusterName: llmos-ceph
configOverride:
# configOverride: |
#   [global]
#   mon_allow_pool_delete = true
#   osd_pool_default_size = 3
#   osd_pool_default_min_size = 2
cephClusterSpec:
  cephVersion:
    image: quay.io/ceph/ceph:v18.2.4
  dataDirHostPath: /var/lib/llmos/rook
  mon:
    count: 3
    allowMultiplePerNode: false
  mgr:
    count: 2
    allowMultiplePerNode: false
  placement:
    all:
      nodeAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 1
          preference:
            matchExpressions:
            - key: role
              operator: In
              values:
              - storage
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/storage
        operator: Exists
  resources:
    mgr:
      limits:
        memory: "1Gi"
      requests:
        cpu: "500m"
        memory: "512Mi"
    mon:
      limits:
        memory: "2Gi"
      requests:
        cpu: "1000m"
        memory: "1Gi"
cephBlockPools:
  - name: llmos-ceph-blockpool
    spec:
      failureDomain: host
      replicated:
        size: 3
      parameters:
        min_size: "1"
`,
				valueContent: `
configOverride: |
  [global]
  mon_allow_pool_delete = true
  osd_pool_default_size = 3
  osd_pool_default_min_size = 2
cephClusterSpec:
  cephVersion:
    image: quay.io/ceph/ceph:v18.2.5
  dataDirHostPath: /var/lib/llmos/rook-test
  mon:
    count: 2
    allowMultiplePerNode: false
  mgr:
    count: 1
    allowMultiplePerNode: false
  placement:
    all:
      nodeAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 1
          preference:
            matchExpressions:
            - key: role
              operator: In
              values:
              - storage
        - weight: 2
          preference:
            matchExpressions:
            - key: role
              operator: In
              values:
              - osd
  resources:
    mgr:
      limits:
        cpu: "1"
        memory: "1Gi"
    mon:
      limits:
        cpu: "2"
        memory: "2Gi"
    osd:
      limits:
        memory: "4Gi"
      requests:
        cpu: "1000m"
        memory: "2Gi"
cephBlockPools:
  - name: llmos-ceph-blockpool
    spec:
      failureDomain: host
      replicated:
        size: 2
      parameters:
        min_size: "1"
        foo: "bar"
`,
			},
			expected: output{
				values: `
operatorNamespace: storage-system
clusterName: llmos-ceph
configOverride: |
  [global]
  mon_allow_pool_delete = true
  osd_pool_default_size = 3
  osd_pool_default_min_size = 2
cephClusterSpec:
  cephVersion:
    image: quay.io/ceph/ceph:v18.2.5
  dataDirHostPath: /var/lib/llmos/rook-test
  mon:
    count: 2
    allowMultiplePerNode: false
  mgr:
    count: 1
    allowMultiplePerNode: false
  placement:
    all:
      nodeAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 1
          preference:
            matchExpressions:
            - key: role
              operator: In
              values:
              - storage
        - weight: 2
          preference:
            matchExpressions:
            - key: role
              operator: In
              values:
              - osd
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/storage
        operator: Exists
  resources:
    mgr:
      limits:
        cpu: "1"
        memory: "1Gi"
      requests:
        cpu: "500m"
        memory: "512Mi"
    mon:
      limits:
        cpu: "2"
        memory: "2Gi"
      requests:
        cpu: "1000m"
        memory: "1Gi"
    osd:
      limits:
        memory: "4Gi"
      requests:
        cpu: "1000m"
        memory: "2Gi"
cephBlockPools:
  - name: llmos-ceph-blockpool
    spec:
      failureDomain: host
      replicated:
        size: 2
      parameters:
        min_size: "1"
        foo: "bar"
`,
				err: nil,
			},
		},
		{
			name: "merge with default values only",
			given: input{
				key: "test",
				defaultValues: `
name: application
replicas: 3
image:
  name: myapp
  tag: v1
`,
				valueContent: "",
			},
			expected: output{
				values: `
name: application
replicas: 3
image:
  name: myapp
  tag: v1
`,
				err: nil,
			},
		},
		{
			name: "merge with values only",
			given: input{
				key:           "test",
				defaultValues: "",
				valueContent: `
replicas: 5
image:
  tag: v2
resources:
  limits:
    memory: "256Mi"
`,
			},
			expected: output{
				values: `
replicas: 5
image:
  tag: v2
resources:
  limits:
    memory: "256Mi"
`,
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		var actual output
		actual.values, actual.err = mergeDefaultValuesContent(tc.given.defaultValues, tc.given.valueContent)
		var (
			result1 = map[string]interface{}{}
			result2 = map[string]interface{}{}
			err     error
		)
		err = yaml.Unmarshal([]byte(actual.values), &result1)
		assert.Nil(t, err)
		err = yaml.Unmarshal([]byte(tc.expected.values), &result2)
		assert.Nil(t, err)
		assert.Equal(t, result1, result2)
		assert.Nil(t, actual.err)
	}
}
