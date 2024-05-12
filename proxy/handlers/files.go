package handlers

import (
	"github.com/sio2project/ft-to-s3/v1/utils"

	"net/http"
)

func Files(w http.ResponseWriter, r *http.Request, logger *utils.LoggerObject, bucketName string) {
	if r.Method == http.MethodPut {
		Put(w, r, logger, bucketName)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
