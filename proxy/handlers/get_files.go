package handlers

import (
	"github.com/sio2project/ft-to-s3/v1/storage"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"io"
	"net/http"
	"strconv"
)

func GetFiles(w http.ResponseWriter, r *http.Request, logger *utils.LoggerObject, bucketName string) {
	path := r.URL.Path[len("/files/"):]
	result := storage.Get(bucketName, logger, path)
	if result.Err != nil {
		logger.Error("Error", result.Err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !result.Found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if result.Gziped {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.Header().Set("Logical-Size", strconv.FormatInt(result.LogicalSize, 10))
	w.Header().Set("Last-Modified", toRFC2822(result.LastModified))
	w.WriteHeader(http.StatusOK)
	_, err := io.Copy(w, result.File)
	if err != nil {
		logger.Error("Error", err)
	}
}
