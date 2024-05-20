package db

func getRefCountName(bucketName string, sha256sum string) string {
	return "ref_count:" + bucketName + ":" + sha256sum
}

func getRefFileName(bucketName string, path string) string {
	return "ref_file:" + bucketName + ":" + path
}

func getModifiedName(bucketName string, path string) string {
	return "modified:" + bucketName + ":" + path
}
