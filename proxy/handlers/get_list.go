package handlers

import (
	"github.com/sio2project/ft-to-s3/v1/storage"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"net/http"
)

func GetList(w http.ResponseWriter, r *http.Request, logger *utils.LoggerObject, bucketName string) {
	path := r.URL.Path[len("/list/"):]
	lastModifiedRFC := r.URL.Query().Get("last_modified")
	if lastModifiedRFC == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("\"?last-modified=\" is required"))
		return
	}
	lastModified, err := FromRFC2822(lastModifiedRFC)
	if err != nil {
		logger.Error("Error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	files, err := storage.GetList(bucketName, logger, path, lastModified)
	if err != nil {
		logger.Error("Error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/text")
	w.WriteHeader(http.StatusOK)
	for _, file := range files {
		w.Write([]byte(file + "\n"))
	}
}
