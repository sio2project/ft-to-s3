package handlers

import (
	"fmt"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"net/http"
)

func Version(w http.ResponseWriter, _ *http.Request, _ *utils.LoggerObject, _ string) {
	set_content_type_json(w)
	_, _ = fmt.Fprint(w, `{"protocol_versions": [2]}`)
}
