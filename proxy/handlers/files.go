package handlers

import (
	"net/http"
	"strings"

	"github.com/sio2project/ft-to-s3/v1/utils"
)

func Files(w http.ResponseWriter, r *http.Request, logger *utils.LoggerObject, bucketName string) {
	if r.Method == http.MethodPut {
		Put(w, r, logger, bucketName)
	} else if r.Method == http.MethodGet {
		if strings.HasPrefix(r.URL.Path, "/files/") {
			GetFiles(w, r, logger, bucketName)
		} else if strings.HasPrefix(r.URL.Path, "/list/") {
			GetList(w, r, logger, bucketName)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
