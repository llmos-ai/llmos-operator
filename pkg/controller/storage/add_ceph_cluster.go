package storage

import (
	"fmt"
	"time"

	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

const (
	CephVersion = "quay.io/ceph/ceph:v18.2.2"

	PCSystemClusterCritical = "system-cluster-critical"
	PCSystemNodeCritical    = "system-node-critical"
)

// setUpDefaultCephCluster helps to set up the default system ceph cluster
func (h *Handler) setUpDefaultCephCluster() error {
	_, err := h.namespaces.Get(constant.CephSystemNamespaceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get system storage namespace %s: %v", constant.CephSystemNamespaceName, err)
	}

	// validate if any ceph cluster already exists
	_, err = h.clusters.Get(constant.CephSystemNamespaceName, constant.CephClusterName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			cephCluster := getDefaultSystemCephCluster()
			if _, err = h.clusters.Create(cephCluster); err != nil {
				return fmt.Errorf("failed to create ceph cluster %s: %v", constant.CephClusterName, err)
			}
			logrus.Infof("creating new system ceph cluster %s/%s", constant.CephSystemNamespaceName, constant.CephClusterName)
			return nil
		}
		return fmt.Errorf("failed to find ceph cluster %s: %v", constant.CephClusterName, err)
	}

	// if ceph cluster already exists, do nothing
	return nil
}

func getDefaultSystemCephCluster() *rookv1.CephCluster {
	cephCluster := &rookv1.CephCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.CephClusterName,
			Namespace: constant.CephSystemNamespaceName,
			Annotations: map[string]string{
				constant.AnnotationAddRookCephRbac:         "true",
				constant.AnnotationAddRookCephBlockStorage: "true",
				constant.AnnotationAddRookCephFilesystem:   "true",
			},
		},
		Spec: rookv1.ClusterSpec{
			DataDirHostPath: "/var/lib/rook",
			Monitoring: rookv1.MonitoringSpec{
				Enabled: false,
			},
			CephVersion: rookv1.CephVersionSpec{
				Image:            CephVersion,
				AllowUnsupported: false,
			},
			Dashboard: rookv1.DashboardSpec{
				Enabled: false,
				SSL:     true,
			},
			LogCollector: rookv1.LogCollectorSpec{
				Enabled:     true,
				MaxLogSize:  resource.NewQuantity(500, resource.DecimalSI),
				Periodicity: "daily",
			},
			Mgr: rookv1.MgrSpec{
				Modules:              []rookv1.Module{},
				Count:                1,
				AllowMultiplePerNode: false,
			},
			Mon: rookv1.MonSpec{
				Count:                1,
				AllowMultiplePerNode: false,
			},
			Storage: rookv1.StorageScopeSpec{
				UseAllNodes: true,
				Selection: rookv1.Selection{
					UseAllDevices: ptr.To(true),
				},
			},
			ContinueUpgradeAfterChecksEvenIfNotHealthy: false,
			CrashCollector: rookv1.CrashCollectorSpec{
				Disable: false,
			},
			CleanupPolicy: rookv1.CleanupPolicySpec{
				AllowUninstallWithVolumes: false,
				Confirmation:              "",
				SanitizeDisks: rookv1.SanitizeDisksSpec{
					Iteration:  1,
					Method:     "quick",
					DataSource: "zero",
				},
			},
			DisruptionManagement: rookv1.DisruptionManagementSpec{
				ManagePodBudgets:      true,
				OSDMaintenanceTimeout: 30,
				PGHealthCheckTimeout:  0,
			},
			Network: rookv1.NetworkSpec{
				Connections: &rookv1.ConnectionsSpec{
					Compression: &rookv1.CompressionSpec{
						Enabled: false,
					},
					Encryption: &rookv1.EncryptionSpec{
						Enabled: false,
					},
					RequireMsgr2: false,
				},
			},
			PriorityClassNames: map[rookv1.KeyType]string{
				rookv1.KeyMgr: PCSystemClusterCritical,
				rookv1.KeyMon: PCSystemNodeCritical,
				rookv1.KeyOSD: PCSystemNodeCritical,
			},
			RemoveOSDsIfOutAndSafeToRemove:    false,
			SkipUpgradeChecks:                 false,
			WaitTimeoutForHealthyOSDInMinutes: time.Duration(10),
			Resources:                         getClusterResources(),
			HealthCheck: rookv1.CephClusterHealthCheckSpec{
				DaemonHealth: rookv1.DaemonHealthSpec{
					Monitor: rookv1.HealthCheckSpec{
						Disabled: false,
						Interval: getTimeDuration(45),
					},
					ObjectStorageDaemon: rookv1.HealthCheckSpec{
						Disabled: false,
						Interval: getTimeDuration(60),
					},
					Status: rookv1.HealthCheckSpec{
						Disabled: false,
						Interval: getTimeDuration(60),
					},
				},
				LivenessProbe: map[rookv1.KeyType]*rookv1.ProbeSpec{
					rookv1.KeyMgr: {
						Disabled: false,
						Probe: &corev1.Probe{
							InitialDelaySeconds: 60,
							PeriodSeconds:       60,
							TimeoutSeconds:      60,
							FailureThreshold:    5,
						},
					},
					rookv1.KeyMon: {
						Disabled: false,
						Probe: &corev1.Probe{
							InitialDelaySeconds: 60,
							PeriodSeconds:       60,
							TimeoutSeconds:      60,
							FailureThreshold:    5,
						},
					},
					rookv1.KeyOSD: {
						Disabled: false,
						Probe: &corev1.Probe{
							InitialDelaySeconds: 60,
							PeriodSeconds:       60,
							TimeoutSeconds:      60,
							FailureThreshold:    5,
						},
					},
				},
			},
		},
	}
	return cephCluster
}

func getTimeDuration(seconds int64) *metav1.Duration {
	return &metav1.Duration{
		Duration: time.Duration(seconds),
	}
}

func getClusterResources() rookv1.ResourceSpec {
	return map[string]corev1.ResourceRequirements{
		"cleanup": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		"crashcollector": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("60Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("60Mi"),
			},
		},
		"exporter": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		"logcollector": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		"mgr": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		"mgr-sidecar": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("40Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		},
		"mon": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		},
		"osd": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
		"prepareosd": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		},
	}
}
