package publicui

import (
	"net/http"

	"github.com/llmos-ai/llmos-controller/pkg/settings"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

type Handler struct {
}

func NewPublicHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	utils.ResponseOKWithBody(rw, map[string]string{
		settings.UIPlSettingName:     settings.UIPl.Get(),
		settings.UISourceSettingName: getUISource(),
	})
}

func getUISource() string {
	uiSource := settings.UISource.Get()
	if uiSource == "auto" {
		if !settings.IsRelease() {
			uiSource = "external"
		} else {
			uiSource = "bundled"
		}
	}
	return uiSource
}
