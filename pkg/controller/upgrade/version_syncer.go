package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	gversion "github.com/hashicorp/go-version"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/slice"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

const (
	syncInterval = time.Hour

	extraInfoClusterUID     = "clusterUID"
	extraInfoNodeCount      = "nodeCount"
	extraInfoCPUCount       = "cpuCount"
	extraInfoMemorySize     = "memorySize"
	extraInfoNvidiaGPUCount = "nvidiaGPUCount"
	checkUrlLabelKey        = "llmos.ai/upgrade-check-url"
)

type CheckUpgradeRequest struct {
	ServerVersion string            `json:"appVersion"`
	ExtraInfo     map[string]string `json:"extraInfo"`
}

type CheckUpgradeResponse struct {
	Versions []Version `json:"versions"`
}

type Version struct {
	Name                 string   `json:"name"` // must be in semantic versioning
	ReleaseDate          string   `json:"releaseDate"`
	KubernetesVersion    string   `json:"kubernetesVersion,omitempty"`
	MinUpgradableVersion string   `json:"minUpgradableVersion,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
}

type versionSyncer struct {
	ctx             context.Context
	httpClient      *retryablehttp.Client
	versionClient   ctlmgmtv1.VersionClient
	versionCache    ctlmgmtv1.VersionCache
	nodeClient      ctlcorev1.NodeClient
	namespaceClient ctlcorev1.NamespaceClient
}

func newVersionSyncer(mgmt *config.Management) *versionSyncer {
	versions := mgmt.MgmtFactory.Management().V1().Version()
	nodes := mgmt.CoreFactory.Core().V1().Node()
	namespaces := mgmt.CoreFactory.Core().V1().Namespace()
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3

	return &versionSyncer{
		ctx:             mgmt.Ctx,
		httpClient:      retryClient,
		versionClient:   versions,
		versionCache:    versions.Cache(),
		nodeClient:      nodes,
		namespaceClient: namespaces,
	}
}

func (s *versionSyncer) start() {
	ticker := time.NewTicker(syncInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.sync(); err != nil {
				logrus.Warnf("failed syncing upgrade versions: %v", err)
			}
		case <-s.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (s *versionSyncer) sync() error {
	checkerEnabled := settings.UpgradeCheckEnabled.Get()
	checkURL := settings.UpgradeCheckUrl.Get()
	if checkerEnabled != "true" || checkURL == "" {
		logrus.Debugf("upgrade checker is disabled or url is empty, skipping upgrade checker")
		return nil
	}
	extraInfo, err := s.getClusterMetaInfo()
	if err != nil {
		return err
	}

	req := &CheckUpgradeRequest{
		ServerVersion: settings.ServerVersion.Get(),
		ExtraInfo:     extraInfo,
	}
	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(req); err != nil {
		return err
	}
	resp, err := s.httpClient.Post(checkURL, "application/json", &requestBody)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Fatalf("failed to close response body: %s", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid upgrade check response, status code: %d", resp.StatusCode)
	}

	var checkResp CheckUpgradeResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return err
	}

	current := settings.ServerVersion.Get()
	return s.syncNewVersions(checkResp, current, checkURL)
}

func (s *versionSyncer) syncNewVersions(resp CheckUpgradeResponse, currentVersion, checkUrl string) error {
	cVersion, err := gversion.NewSemver(currentVersion)
	if err != nil {
		return err
	}

	for _, v := range resp.Versions {
		newVersion := &mgmtv1.Version{
			ObjectMeta: metav1.ObjectMeta{
				Name: v.Name,
				Labels: map[string]string{
					checkUrlLabelKey: checkUrl,
				},
			},
			Spec: mgmtv1.VersionSpec{
				ReleaseDate:          v.ReleaseDate,
				KubernetesVersion:    v.KubernetesVersion,
				MinUpgradableVersion: v.MinUpgradableVersion,
				Tags:                 v.Tags,
			},
		}
		canUpgrade, err := canUpgrade(cVersion, newVersion)
		if err != nil {
			logrus.Debugf("failed to compare version %s with current version %s: %v", v.Name, currentVersion, err)
			continue
		}

		if !canUpgrade {
			continue
		}
		foundVersion, err := s.versionCache.Get(newVersion.Name)
		if err != nil && apierrors.IsNotFound(err) {
			if _, err = s.versionClient.Create(newVersion); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		if foundVersion != nil && !reflect.DeepEqual(foundVersion.Spec, newVersion.Spec) {
			toUpdate := foundVersion.DeepCopy()
			toUpdate.Spec = newVersion.Spec
			if _, err = s.versionClient.Update(toUpdate); err != nil {
				return err
			}
		}
	}

	return s.cleanupVersions(cVersion)
}

func canUpgrade(currentVersion *gversion.Version, newVersion *mgmtv1.Version) (bool, error) {
	nVersion, err := gversion.NewSemver(newVersion.Name)
	if err != nil {
		return false, err
	}

	// Validate Kubernetes version if it's set
	if newVersion.Spec.KubernetesVersion != "" {
		if _, err = gversion.NewSemver(newVersion.Spec.KubernetesVersion); err != nil {
			return false, err
		}
	}

	miniUpgradeableVersion, err := gversion.NewSemver(newVersion.Spec.MinUpgradableVersion)
	if err != nil {
		return false, err
	}

	switch {
	case isDevVersion(newVersion.Name, newVersion.Spec.Tags):
		return true, nil
	case nVersion.GreaterThan(currentVersion) &&
		(newVersion.Spec.MinUpgradableVersion == "" || currentVersion.GreaterThanOrEqual(miniUpgradeableVersion)):
		return true, nil
	default:
		return false, nil
	}
}

// getClusterMetaInfo returns the cluster info for telemetry
func (s *versionSyncer) getClusterMetaInfo() (map[string]string, error) {
	nodes, err := s.nodeClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	systemNs, err := s.namespaceClient.Get(constant.SystemNamespaceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	extraInfo := map[string]string{}
	extraInfo[extraInfoClusterUID] = string(systemNs.UID)
	extraInfo[extraInfoNodeCount] = strconv.Itoa(len(nodes.Items))

	cpu := resource.NewQuantity(0, resource.BinarySI)
	memory := resource.NewQuantity(0, resource.BinarySI)
	gpu := resource.NewQuantity(0, resource.DecimalExponent)
	for _, node := range nodes.Items {
		cpu.Add(*node.Status.Capacity.Cpu())
		memory.Add(*node.Status.Capacity.Memory())
		gpu.Add(node.Status.Capacity[constant.NvidiaGPUKey])
	}
	extraInfo[extraInfoCPUCount] = cpu.String()
	extraInfo[extraInfoMemorySize] = convertToGi(memory)
	extraInfo[extraInfoNvidiaGPUCount] = gpu.String()
	logrus.Debugf("get cluster info: %v", extraInfo)
	return extraInfo, nil
}

func convertToGi(q *resource.Quantity) string {
	giValue := float64(q.Value()) / math.Pow(1024, 3) // Convert bytes to Gi
	if giValue < 1 {
		return fmt.Sprintf("%dMi", int64(math.Ceil(giValue*1024)))
	}
	return fmt.Sprintf("%dGi", int64(math.Ceil(giValue)))
}

// cleanupVersions removes old versions that are no longer upgradable
func (s *versionSyncer) cleanupVersions(currentVersion *gversion.Version) error {
	versions, err := s.versionCache.List(labels.Everything())
	if err != nil {
		return err
	}

	for _, v := range versions {
		version, err := gversion.NewSemver(v.Name)
		if err != nil {
			return fmt.Errorf("failed to parse version %s: %v", v.Name, err)
		}

		if currentVersion.GreaterThanOrEqual(version) && !isDevVersion(v.Name, v.Spec.Tags) {
			logrus.Infof("removing old version %s", v.Name)
			if err = s.versionClient.Delete(v.Name, &metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func isDevVersion(version string, tags []string) bool {
	return strings.Contains(version, "dev") || slice.ContainsString(tags, "dev")
}
