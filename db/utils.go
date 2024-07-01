package db

func GetRefCountName(bucketName string, sha256sum string) string {
	return "ref_count:" + bucketName + ":" + sha256sum
}

func GetRefFileName(bucketName string, path string) string {
	return "ref_file:" + bucketName + ":" + path
}

func GetModifiedName(bucketName string, path string) string {
	return "modified:" + bucketName + ":" + path
}
