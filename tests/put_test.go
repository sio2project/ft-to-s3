package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/proxy/handlers"
	"github.com/sio2project/ft-to-s3/v1/storage"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"github.com/stretchr/testify/assert"
)

// Configuration:

func TestMain(m *testing.M) {
	setup()
	m.Run()
	destroy()
}

// Helpers:

func testCustomRefCount(t *testing.T, sha256digest string, path string, content string, modified string, contentGzipped bool,
	refCount int) {
	refCountdb, err := db.GetRefCount("test", sha256digest)
	assert.NoError(t, err)
	assert.Equal(t, refCount, refCountdb)

	dbmodified, err := db.GetModified("test", path)
	assert.NoError(t, err)
	timestamp, err := handlers.FromRFC2822(modified)
	assert.NoError(t, err)
	assert.Equal(t, timestamp, dbmodified)

	hash, err := db.GetHashForPath("test", path)
	assert.NoError(t, err)
	assert.Equal(t, sha256digest, hash)

	minioClient := storage.GetClient()
	object, err := minioClient.GetObject(context.Background(), "test", sha256digest,
		minio.GetObjectOptions{})
	assert.NoError(t, err)
	var data []byte
	if contentGzipped {
		reader, err := gzip.NewReader(object)
		assert.NoError(t, err)
		data, err = io.ReadAll(reader)
		assert.NoError(t, err)
	} else {
		data, err = io.ReadAll(object)
		assert.NoError(t, err)
	}
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func testStandard(t *testing.T, sha256digest string, path string, content string, modified string, contentGzipped bool) {
	testCustomRefCount(t, sha256digest, path, content, modified, contentGzipped, 1)
}

func putFile(path string, content []byte, lastModified string, headers map[string]string) (int, string) {
	status, body := makeRequest("http://localhost:8080/files/"+path,
		http.MethodPut,
		content,
		map[string]string{
			"last_modified": lastModified,
		},
		headers,
	)
	return status, body
}

func putFileString(path string, content string, lastModified string, headers map[string]string) (int, string) {
	return putFile(path, []byte(content), lastModified, headers)
}

func PutFileCalculateHeaders(path string, content string, lastModified string) (int, string) {
	sha256digest := utils.Sha256Checksum([]byte(content))
	return putFileString(path, content, lastModified, map[string]string{
		"Sha256-Checksum": sha256digest,
		"Logical-Size":    strconv.Itoa(len(content)),
	})
}

// Tests:

func TestPut(t *testing.T) {
	clean()

	status, _ := putFileString("test_file", "test file content", "Mon, 20 May 2024 15:04:05 MST", map[string]string{})
	assert.Equal(t, http.StatusOK, status)

	sha256digest := utils.Sha256Checksum([]byte("test file content"))
	testStandard(t, sha256digest, "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)

	clean()
	status, _ = PutFileCalculateHeaders("test_file", "test file content", "Mon, 20 May 2024 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, sha256digest, "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)
}

func TestWithHeaders(t *testing.T) {
	clean()

	sha256digest := utils.Sha256Checksum([]byte("test file content"))
	var content bytes.Buffer
	gzipWriter := gzip.NewWriter(&content)
	_, err := gzipWriter.Write([]byte("test file content"))
	assert.NoError(t, err)
	err = gzipWriter.Close()
	assert.NoError(t, err)

	status, _ := putFile("test_file", content.Bytes(), "Mon, 20 May 2024 15:04:05 MST",
		map[string]string{
			"Content-Encoding": "gzip",
			"Sha256-Checksum":  sha256digest,
			"Logical-Size":     strconv.Itoa(len("test file content")),
		},
	)
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, sha256digest, "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", true)
}

func TestFileExists(t *testing.T) {
	clean()

	status, _ := PutFileCalculateHeaders("test_file", "test file content", "Mon, 20 May 2024 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, utils.Sha256Checksum([]byte("test file content")), "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)

	// Test if older last modified date doesn't overwrite the file
	status, _ = PutFileCalculateHeaders("test_file", "test file content", "Mon, 20 May 2000 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, utils.Sha256Checksum([]byte("test file content")), "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)
}

func TestFileOverwrite(t *testing.T) {
	clean()

	status, _ := PutFileCalculateHeaders("test_file", "test file content", "Mon, 20 May 2024 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, utils.Sha256Checksum([]byte("test file content")), "test_file", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)

	// Overwrite the file by specifying a newer last modified date
	status, _ = PutFileCalculateHeaders("test_file", "test file content 2", "Mon, 20 May 2124 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, utils.Sha256Checksum([]byte("test file content 2")), "test_file", "test file content 2",
		"Mon, 20 May 2124 15:04:05 MST", false)

	// TODO: Check if the old file is deleted (requires working DELETE)
}

func TestMultpileFilesSameHash(t *testing.T) {
	clean()

	status, _ := PutFileCalculateHeaders("test_file1", "test file content", "Mon, 20 May 2024 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testStandard(t, utils.Sha256Checksum([]byte("test file content")), "test_file1", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false)

	status, _ = PutFileCalculateHeaders("test_file2", "test file content", "Mon, 20 May 2024 15:04:05 MST")
	assert.Equal(t, http.StatusOK, status)
	testCustomRefCount(t, utils.Sha256Checksum([]byte("test file content")), "test_file2", "test file content",
		"Mon, 20 May 2024 15:04:05 MST", false, 2)
}
