package handlers

import (
	"fmt"
	"net/http"

	"github.com/sio2project/ft-to-s3/v1/utils"
)

func Version(w http.ResponseWriter, _ *http.Request, _ *utils.Logger) {
	set_content_type_json(w)
	_, _ = fmt.Fprint(w, `{"protocol_versions": [2]}`)
}
