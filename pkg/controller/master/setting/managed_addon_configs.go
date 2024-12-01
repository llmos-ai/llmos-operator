package setting

import (
	"encoding/json"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

type ManagedAddonConfigs struct {
	LLMOSMonitoring LLMOSMonitoring `json:"llmos-monitoring"`
	LLMOSGPUStack   LLMOSGPUStack   `json:"llmos-gpu-stack"`
}

type LLMOSMonitoring struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type LLMOSGPUStack struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

func (h *handler) setManagedAddonConfigs(setting *mgmtv1.Setting) error {
	val, err := h.constructedManagedAddonConfigs()
	if err != nil {
		return err
	}
	return h.Set(setting.Name, val)
}

func (h *handler) constructedManagedAddonConfigs() (string, error) {
	monitoring, err := h.managedAddonCache.Get(constant.MonitoringNamespace, "llmos-monitoring")
	if err != nil {
		return "", err
	}

	gpuStack, err := h.managedAddonCache.Get(constant.SystemNamespaceName, "llmos-gpu-stack")
	if err != nil {
		return "", err
	}

	cfg := ManagedAddonConfigs{
		LLMOSMonitoring: LLMOSMonitoring{
			Enabled: monitoring.Spec.Enabled,
			Status:  string(monitoring.Status.State),
		},
		LLMOSGPUStack: LLMOSGPUStack{
			Enabled: gpuStack.Spec.Enabled,
			Status:  string(gpuStack.Status.State),
		},
	}

	cfgStr, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(cfgStr), nil
}

func DecodeManagedAddonConfigs(val string) (*ManagedAddonConfigs, error) {
	cfg := &ManagedAddonConfigs{}
	err := json.Unmarshal([]byte(val), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
