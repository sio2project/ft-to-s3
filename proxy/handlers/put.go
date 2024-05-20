package handlers

import (
	"github.com/sio2project/ft-to-s3/v1/storage"
	"github.com/sio2project/ft-to-s3/v1/utils"

	"net/http"
	"strconv"
)

func Put(w http.ResponseWriter, r *http.Request, logger *utils.LoggerObject, bucketName string) {
	path := r.URL.Path[len("/files/"):]

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

	compressed := r.Header.Get("Content-Encoding") == "gzip"
	digest := r.Header.Get("Sha256-Checksum")
	contentLengthStr := r.Header.Get("Content-Length")
	var contentLength int64 = -1
	if contentLengthStr != "" {
		contentLength, err = strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			logger.Error("Error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	logicalSizeStr := r.Header.Get("Logical-Size")
	var logicalSize int64 = -1
	if logicalSizeStr != "" {
		logicalSize, err = strconv.ParseInt(logicalSizeStr, 10, 64)
		if err != nil {
			logger.Error("Error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	version, err := storage.Store(bucketName, logger, path, r.Body, lastModified, contentLength,
		compressed, digest, logicalSize)

	if err != nil {
		logger.Error("Error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Last-Modified", toRFC2822(version))
}
