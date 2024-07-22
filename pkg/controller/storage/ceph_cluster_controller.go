package storage

import (
	"bytes"
	"context"
	"fmt"

	ctlappsv1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1"
	ctlrbacv1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/rbac/v1"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlrookv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ceph.rook.io/v1"
	ctlstoragev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/template"
)

type Handler struct {
	clusters             ctlrookv1.CephClusterClient
	clusterCache         ctlrookv1.CephClusterCache
	blockPools           ctlrookv1.CephBlockPoolClient
	fileSystems          ctlrookv1.CephFilesystemClient
	fsSubVolGroups       ctlrookv1.CephFilesystemSubVolumeGroupClient
	namespaces           ctlcorev1.NamespaceClient
	namespaceCache       ctlcorev1.NamespaceCache
	serviceAccounts      ctlcorev1.ServiceAccountClient
	roles                ctlrbacv1.RoleClient
	clusterRolesBindings ctlrbacv1.ClusterRoleBindingClient
	roleBindings         ctlrbacv1.RoleBindingClient
	storageClasses       ctlstoragev1.StorageClassClient
	deployments          ctlappsv1.DeploymentClient
	deploymentCache      ctlappsv1.DeploymentCache
}

const (
	cephClusterOnChange = "cephCluster.onChange"
	strTrue             = "true"

	cephClusterSATemplate          = "ceph-cluster-sa.yaml"
	cephClusterCRBTemplate         = "ceph-cluster-crb.yaml"
	cephClusterRoleTemplate        = "ceph-cluster-role.yaml"
	cephClusterRoleBindingTemplate = "ceph-cluster-role-binding.yaml"
	cephBlockPoolTemplate          = "ceph-block-pool.yaml"
	cephBlockPoolSCTemplate        = "ceph-block-pool-sc.yaml"
	cephFilesystemTemplate         = "ceph-filesystem.yaml"
	cephFilesystemSCTemplate       = "ceph-fs-sc.yaml"
	cephFsSubVolumeGroupTemplate   = "ceph-fs-subvolgroup.yaml"
	cephToolboxTemplate            = "ceph-toolbox.yaml"
)

func Register(ctx context.Context, mgmt *config.Management) error {
	rook := mgmt.RookFactory.Ceph().V1()
	cephCluster := rook.CephCluster()
	namespace := mgmt.CoreFactory.Core().V1().Namespace()
	sas := mgmt.CoreFactory.Core().V1().ServiceAccount()
	rbac := mgmt.RbacFactory.Rbac().V1()
	storage := mgmt.StorageFactory.Storage().V1().StorageClass()
	deployments := mgmt.AppsFactory.Apps().V1().Deployment()
	h := Handler{
		clusters:             cephCluster,
		clusterCache:         cephCluster.Cache(),
		blockPools:           rook.CephBlockPool(),
		fileSystems:          rook.CephFilesystem(),
		fsSubVolGroups:       rook.CephFilesystemSubVolumeGroup(),
		namespaces:           namespace,
		namespaceCache:       namespace.Cache(),
		serviceAccounts:      sas,
		roles:                rbac.Role(),
		clusterRolesBindings: rbac.ClusterRoleBinding(),
		roleBindings:         rbac.RoleBinding(),
		storageClasses:       storage,
		deployments:          deployments,
		deploymentCache:      deployments.Cache(),
	}

	cephCluster.OnChange(ctx, cephClusterOnChange, h.OnChanged)

	return h.setUpDefaultCephCluster()
}

func (h *Handler) OnChanged(_ string, cluster *rookv1.CephCluster) (*rookv1.CephCluster, error) {
	if cluster == nil || cluster.DeletionTimestamp != nil {
		return nil, nil
	}
	if needToAddCephClusterDependencies(cluster) {
		cephConfig := template.NewCephConfig(cluster.Name, cluster.Namespace, constant.SystemNamespaceName)
		ownerRefers := getClusterOwnerReference(cluster)

		// update cluster condition of applied rbac
		clusterCpy := cluster.DeepCopy()
		if cluster.Annotations[constant.AnnotationAddRookCephRbac] == strTrue &&
			cluster.Annotations[constant.AnnotationAddedRookCephRbac] != strTrue {
			if err := h.applyCephClusterRBAC(cephConfig, ownerRefers); err != nil {
				return cluster, err
			}
			clusterCpy.Annotations[constant.AnnotationAddedRookCephRbac] = strTrue
		}

		if cluster.Annotations[constant.AnnotationAddRookCephBlockStorage] == strTrue &&
			cluster.Annotations[constant.AnnotationAddedRookCephBlockStorage] != strTrue {
			if err := h.applyCephBlockStorage(cephConfig, ownerRefers); err != nil {
				return cluster, err
			}
			clusterCpy.Annotations[constant.AnnotationAddedRookCephBlockStorage] = strTrue
		}

		if cluster.Annotations[constant.AnnotationAddRookCephFilesystem] == strTrue &&
			cluster.Annotations[constant.AnnotationAddedRookCephFilesystem] != strTrue {
			if err := h.applyCephFilesystem(cephConfig, ownerRefers); err != nil {
				return cluster, err
			}
			clusterCpy.Annotations[constant.AnnotationAddedRookCephFilesystem] = strTrue
		}

		if cluster.Annotations[constant.AnnotationAddCephToolbox] == strTrue &&
			cluster.Annotations[constant.AnnotationAddedCephToolbox] != strTrue {
			if err := h.addCephToolbox(cluster, cephConfig, ownerRefers); err != nil {
				return cluster, err
			}
			clusterCpy.Annotations[constant.AnnotationAddedCephToolbox] = strTrue
		}

		// update cluster condition of applied ceph block storage
		if _, err := h.clusters.Update(clusterCpy); err != nil {
			return cluster, err
		}
	}

	return cluster, nil
}

func needToAddCephClusterDependencies(cluster *rookv1.CephCluster) bool {
	annos := cluster.Annotations
	if annos == nil {
		return false
	}

	if cluster.Annotations[constant.AnnotationAddRookCephRbac] == strTrue &&
		cluster.Annotations[constant.AnnotationAddedRookCephRbac] != strTrue {
		return true
	}

	if cluster.Annotations[constant.AnnotationAddRookCephBlockStorage] == strTrue &&
		cluster.Annotations[constant.AnnotationAddedRookCephBlockStorage] != strTrue {
		return true
	}

	if cluster.Annotations[constant.AnnotationAddRookCephFilesystem] == strTrue &&
		cluster.Annotations[constant.AnnotationAddedRookCephFilesystem] != strTrue {
		return true
	}

	if cluster.Annotations[constant.AnnotationAddCephToolbox] == strTrue &&
		cluster.Annotations[constant.AnnotationAddedCephToolbox] != strTrue {
		return true
	}

	return false
}

func (h *Handler) applyCephClusterRBAC(cfg *template.CephConfig, refs []metav1.OwnerReference) error {
	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephClusterSATemplate,
		&corev1.ServiceAccount{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephClusterRoleTemplate,
		&rbacv1.Role{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephClusterRoleBindingTemplate,
		&rbacv1.RoleBinding{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephClusterCRBTemplate,
		&rbacv1.ClusterRoleBinding{}); err != nil {
		return err
	}
	return nil
}

func (h *Handler) applyCephBlockStorage(cfg *template.CephConfig, refs []metav1.OwnerReference) error {
	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephBlockPoolTemplate,
		&rookv1.CephBlockPool{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephBlockPoolSCTemplate,
		&v1.StorageClass{}); err != nil {
		return err
	}
	return nil
}

func (h *Handler) applyCephFilesystem(cfg *template.CephConfig, refs []metav1.OwnerReference) error {
	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephFilesystemTemplate,
		&rookv1.CephFilesystem{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephFsSubVolumeGroupTemplate,
		&rookv1.CephFilesystemSubVolumeGroup{}); err != nil {
		return err
	}

	if err := h.applyCephClusterCustomTemplate(cfg, refs, cephFilesystemSCTemplate,
		&v1.StorageClass{}); err != nil {
		return err
	}
	return nil
}

func getClusterOwnerReference(cluster *rookv1.CephCluster) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion: cluster.APIVersion,
			Kind:       cluster.Kind,
			Name:       cluster.Name,
			UID:        cluster.UID,
		},
	}
}

// nolint: gocyclo
func (h *Handler) applyCephClusterCustomTemplate(cfg *template.CephConfig,
	refs []metav1.OwnerReference, fileName string, obj interface{}) error {
	if cfg == nil {
		return fmt.Errorf("cephConfig shoud not be nil")
	}

	if len(refs) == 0 {
		return fmt.Errorf("owner reference is empty")
	}

	templates, err := template.Render(fileName, cfg)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	yamls := bytes.Split(templates.Bytes(), []byte("\n---\n"))
	for _, yml := range yamls {
		if len(yml) == 0 {
			continue
		}

		switch obj.(type) {
		case *corev1.ServiceAccount:
			sa := &corev1.ServiceAccount{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(sa); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			sa.SetOwnerReferences(refs)
			if _, err = h.serviceAccounts.Create(sa); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create sa %s/%s", sa.Namespace, sa.Name)
			}
		case *rbacv1.ClusterRoleBinding:
			crb := &rbacv1.ClusterRoleBinding{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(crb); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			crb.SetOwnerReferences(refs)
			if _, err = h.clusterRolesBindings.Create(crb); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create clusterRoleBinding %s/%s", crb.Namespace, crb.Name)
			}
		case *rbacv1.Role:
			role := &rbacv1.Role{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(role); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			role.SetOwnerReferences(refs)
			if _, err = h.roles.Create(role); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create role %s/%s", role.Namespace, role.Name)
			}
		case *rbacv1.RoleBinding:
			rb := &rbacv1.RoleBinding{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(rb); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			rb.SetOwnerReferences(refs)
			if _, err = h.roleBindings.Create(rb); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create roleBinding %s/%s", rb.Namespace, rb.Name)
			}
		case *v1.StorageClass:
			sc := &v1.StorageClass{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(sc); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			sc.SetOwnerReferences(refs)
			if _, err = h.storageClasses.Create(sc); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create storageClass %s/%s", sc.Namespace, sc.Name)
			}
		case *rookv1.CephBlockPool:
			cBlock := &rookv1.CephBlockPool{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(cBlock); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			cBlock.SetOwnerReferences(refs)
			if _, err = h.blockPools.Create(cBlock); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create block pool %s/%s", cBlock.Namespace, cBlock.Name)
			}
		case *rookv1.CephFilesystem:
			fs := &rookv1.CephFilesystem{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(fs); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			fs.SetOwnerReferences(refs)
			if _, err = h.fileSystems.Create(fs); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create filesystem %s/%s", fs.Namespace, fs.Name)
			}
		case *rookv1.CephFilesystemSubVolumeGroup:
			fsGroup := &rookv1.CephFilesystemSubVolumeGroup{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(fsGroup); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			fsGroup.SetOwnerReferences(refs)
			if _, err = h.fsSubVolGroups.Create(fsGroup); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create filesystem %s/%s", fsGroup.Namespace, fsGroup.Name)
			}
		case *appsv1.Deployment:
			dep := &appsv1.Deployment{}
			if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 1024).Decode(dep); err != nil {
				return fmt.Errorf("failed to decode yaml, error: %s", err.Error())
			}

			dep.SetOwnerReferences(refs)
			if _, err = h.deployments.Create(dep); err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create filesystem %s/%s", dep.Namespace, dep.Name)
			}
		default:
			return fmt.Errorf("unsupported object type")
		}
	}

	return nil
}

func (h *Handler) addCephToolbox(cluster *rookv1.CephCluster,
	cfg *template.CephConfig, refs []metav1.OwnerReference) error {
	_, err := h.deploymentCache.Get(cluster.Namespace, cfg.GetToolboxName())
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err = h.applyCephClusterCustomTemplate(cfg, refs, cephToolboxTemplate,
				&appsv1.Deployment{}); err != nil {
				return err
			}
		}
		return err
	}

	return nil
}
