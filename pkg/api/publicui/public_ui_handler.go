package publicui

import (
	"net/http"

	"github.com/llmos-ai/llmos-operator/pkg/server/ui"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

type Handler struct {
}

func NewPublicHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	utils.ResponseOKWithBody(rw, map[string]string{
		settings.UIPlSettingName:                  settings.UIPl.Get(),
		settings.UISourceSettingName:              getUISource(),
		settings.DefaultNotebookImagesSettingName: settings.DefaultNotebookImages.Get(),
		settings.FirstLoginSettingName:            settings.FirstLogin.Get(),
	})
}

func getUISource() string {
	uiSource := settings.UISource.Get()
	if uiSource == ui.SourceAuto {
		if !settings.IsRelease() {
			uiSource = ui.SourceExternal
		} else {
			uiSource = ui.SourceBundle
		}
	}
	return uiSource
}
