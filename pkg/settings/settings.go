package settings

// referred to the code of https://github.com/harvester/harvester/blob/master/pkg/settings/settings.go
import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	settings       = map[string]Setting{}
	provider       Provider
	InjectDefaults string
	releasePattern = regexp.MustCompile("^v[0-9]")

	// APIUIVersion also update the API_UI_VERSION in Dockerfile
	APIUIVersion                 = NewSetting("api-ui-version", "1.1.11")
	AuthUserSessionMaxTTLMinutes = NewSetting("auth-user-session-max-ttl-minutes", "720") // 12 hrs
	AuthTokenMaxTTLMinutes       = NewSetting("auth-token-max-ttl-minutes", "129600")     // 90 days
	FirstLogin                   = NewSetting(FirstLoginSettingName, "true")
	ServerURL                    = NewSetting("server-url", "")
	ServerVersion                = NewSetting(ServerVersionName, "dev")
	UIIndex                      = NewSetting("ui-index", "https://releases.1block.ai/dashboard/latest/index.html")
	UIPath                       = NewSetting("ui-path", "/usr/share/llmos/llmos-operator")
	UIPl                         = NewSetting(UIPlSettingName, "llmos")    // UIPl is the private vendor/company name
	UISource                     = NewSetting(UISourceSettingName, "auto") // Options are 'auto', 'external' or 'bundled'
	// LocalLLMServerURL specify local LLM server url, e.g., http://llmos-ollama.llmos-system:11434
	LocalLLMServerURL = NewSetting(LocalLLMServerURLSettingName, "")
	// DatabaseURL set local database url, e.g., postgresql://user:password@llmos-postgresql.llmos-system:5432/llmos
	DatabaseURL           = NewSetting(DatabaseUrlSettingName, "")
	DefaultNotebookImages = NewSetting(DefaultNotebookImagesSettingName, setDefaultNotebookImages())
	UpgradeCheckEnabled   = NewSetting(UpgradeCheckEnabledName, "true")
	UpgradeCheckUrl       = NewSetting(UpgradeCheckUrlName, "https://llmos-upgrade.1block.ai/v1/versions")
	LLMOSDefaultRegistry  = NewSetting(LLMOSDefaultRegistryName, "docker.io/llmosai")
	LogLevel              = NewSetting(LogLevelSettingName, "info") // options are info, debug and trace
	ManagedAddonConfigs   = NewSetting(ManagedAddonConfigsName, "")
)

const (
	UIPlSettingName                  = "ui-pl"
	UISourceSettingName              = "ui-source"
	FirstLoginSettingName            = "first-login"
	LocalLLMServerURLSettingName     = "local-llm-server-url"
	DatabaseUrlSettingName           = "database-url"
	DefaultNotebookImagesSettingName = "default-notebook-images"
	ServerVersionName                = "server-version"
	UpgradeCheckEnabledName          = "upgrade-check-enabled"
	UpgradeCheckUrlName              = "upgrade-check-url"
	LLMOSDefaultRegistryName         = "llmos-default-registry"
	LogLevelSettingName              = "log-level"
	ManagedAddonConfigsName          = "managed-addon-configs"
)

func init() {
	if InjectDefaults == "" {
		return
	}
	defaults := map[string]string{}
	if err := json.Unmarshal([]byte(InjectDefaults), &defaults); err != nil {
		return
	}
	for name, defaultValue := range defaults {
		value, ok := settings[name]
		if !ok {
			continue
		}
		value.Default = defaultValue
		settings[name] = value
	}
}

type Provider interface {
	Get(name string) string
	Set(name, value string) error
	SetIfUnset(name, value string) error
	SetAll(settings map[string]Setting) error
}

type Setting struct {
	Name     string
	Default  string
	ReadOnly bool
}

func (s Setting) SetIfUnset(value string) error {
	if provider == nil {
		return s.Set(value)
	}
	return provider.SetIfUnset(s.Name, value)
}

func (s Setting) Set(value string) error {
	if provider == nil {
		s, ok := settings[s.Name]
		if ok {
			s.Default = value
			settings[s.Name] = s
		}
	} else {
		return provider.Set(s.Name, value)
	}
	return nil
}

func (s Setting) Get() string {
	if provider == nil {
		s := settings[s.Name]
		return s.Default
	}
	return provider.Get(s.Name)
}

func (s Setting) GetInt() int {
	v := s.Get()
	i, err := strconv.Atoi(v)
	if err == nil {
		return i
	}
	logrus.Errorf("failed to parse setting %s=%s as int: %v", s.Name, v, err)
	i, err = strconv.Atoi(s.Default)
	if err != nil {
		return 0
	}
	return i
}

func SetProvider(p Provider) error {
	if err := p.SetAll(settings); err != nil {
		return err
	}
	provider = p
	return nil
}

func NewSetting(name, def string) Setting {
	s := Setting{
		Name:    name,
		Default: def,
	}
	settings[s.Name] = s
	return s
}

func GetEnvKey(key string) string {
	return "LLMOS_" + strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func IsRelease() bool {
	return !strings.Contains(ServerVersion.Get(), "head") && releasePattern.MatchString(ServerVersion.Get())
}
