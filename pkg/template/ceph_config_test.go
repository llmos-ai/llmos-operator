package template_test

import (
	"bytes"
	"testing"

	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/llmos-ai/llmos-operator/pkg/template"
)

var cephTemplates = map[string]interface{}{
	"ceph-cluster-sa.yaml":           &corev1.ServiceAccount{},
	"ceph-cluster-crb.yaml":          &rbacv1.ClusterRoleBinding{},
	"ceph-cluster-role.yaml":         &rbacv1.Role{},
	"ceph-cluster-role-binding.yaml": &rbacv1.RoleBinding{},
	"ceph-block-pool.yaml":           &rookv1.CephBlockPool{},
	"ceph-block-pool-sc.yaml":        &v1.StorageClass{},
	"ceph-filesystem.yaml":           rookv1.CephFilesystem{},
	"ceph-fs-sc.yaml":                &v1.StorageClass{},
	"ceph-fs-subvolgroup.yaml":       &rookv1.CephFilesystemSubVolumeGroup{},
	"ceph-toolbox.yaml":              &appsv1.Deployment{},
}

const (
	cephClusterName = "llmos-ceph"
	cephNamespace   = "ceph-system"
	systemNamespace = "llmos-system"
)

func Test_NewCephConfig(t *testing.T) {
	cfg := template.NewCephConfig(cephClusterName, cephNamespace, systemNamespace)
	for tmp, obj := range cephTemplates {
		templates, err := template.Render(template.CephClusterTemplate, tmp, cfg)
		assert.NoError(t, err, "expect no error during template rendering")
		yamls := bytes.Split(templates.Bytes(), []byte("\n---\n"))
		for _, yml := range yamls {
			if len(yml) == 0 {
				continue
			}

			switch obj.(type) {
			case *corev1.ServiceAccount:
				objDec := &corev1.ServiceAccount{}
				err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(objDec)
				assert.NoError(t, err, "expect no error during yaml decoding")
				assert.Equal(t, objDec.Namespace, cephNamespace)
			case *rbacv1.Role:
				objDec := &rbacv1.Role{}
				err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(objDec)
				assert.NoError(t, err, "expect no error during yaml decoding")
				assert.Equal(t, objDec.Namespace, cephNamespace)
			case *rookv1.CephBlockPool:
				objDec := &rookv1.CephBlockPool{}
				err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(objDec)
				assert.NoError(t, err, "expect no error during yaml decoding")
				assert.Equal(t, objDec.Spec.Replicated.Size, uint(2))
				assert.Equal(t, objDec.Spec.Parameters["min_size"], "1")
			case *rookv1.CephFilesystem:
				objDec := &rookv1.CephFilesystem{}
				err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(objDec)
				assert.NoError(t, err, "expect no error during yaml decoding")
				assert.Equal(t, objDec.Namespace, cephNamespace)
				assert.Equal(t, objDec.Spec.DataPools[0], rookv1.NamedPoolSpec{
					Name: cfg.FilesystemName(),
					PoolSpec: rookv1.PoolSpec{
						Replicated: rookv1.ReplicatedSpec{
							Size: 2,
						},
					},
				})
				assert.Equal(t, objDec.Spec.MetadataPool.Replicated.Size, uint(1))
			default:
				err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(obj)
				assert.NoError(t, err, "expect no error during yaml decoding")
			}
		}
	}

}
