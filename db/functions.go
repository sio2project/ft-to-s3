package db

import "github.com/redis/go-redis/v9"

func CanUploadFile(bucketName string, hash string) (int, error) {
	function := redis.NewScript(`
		local bucketName = KEYS[1]
		local hash = KEYS[2]
		ref_count = redis.call('GET', 'ref_count:' .. bucketName .. ':' .. hash) -- Check if the file exists
		if ref_count == nil then  -- The file does not exist, set ref_count to 0 to indicate that the file is being processed.
			redis.call('SET', 'ref_count:' .. bucketName .. ':' .. hash, 0, 'EX', 60) -- Set the timeout in case the client crashes.
			return 1
		end
		if ref_count == 0 then
			return 0
		end
		return 2
	`)

	return function.Run(redisContext, rdb, []string{bucketName, hash}).Int()
}

func AddFile(bucketName string, hash string, path string, last_modified int) error {
	function := redis.NewScript(`
		local bucketName = KEYS[1]
		local hash = KEYS[2]
		local path = KEYS[3]
		local last_modified = ARGV[1]
		redis.call('INCR', 'ref_count:' .. bucketName .. ':' .. hash)
		redis.call('SET', 'ref_file:' .. bucketName .. ':' .. path, hash)
		redis.call('SET', 'modified:' .. bucketName .. ':' .. path, last_modified)
	`)
	return function.Run(redisContext, rdb, []string{bucketName, hash, path}, last_modified).Err()
}

func CanDeleteFile(bucketName string, hash string) (int, error) {
	function := redis.NewScript(`
		local bucketName = KEYS[1]
		local hash = KEYS[2]
		ref_count = redis.call('DECR', 'ref_count:' .. bucketName .. ':' .. hash)
		if ref_count == 0 then -- The file is not referenced anymore, set the ref_count to 0 to indicate that the file is being processed.
			redis.call('EXPIRE', 'ref_count:' .. bucketName .. ':' .. hash, 60) -- Set the timeout in case the client crashes.
			return 1
		end
		return 0
	`)
	return function.Run(redisContext, rdb, []string{bucketName, hash}).Int()
}

func DeleteFile(bucketName string, hash string, path string, last bool) error {
	var function *redis.Script
	if last {
		function = redis.NewScript(`
			local bucketName = KEYS[1]
			local hash = KEYS[2]
			local path = KEYS[3]
			redis.call('DEL', 'ref_count:' .. bucketName .. ':' .. hash)
			redis.call('DEL', 'ref_file:' .. bucketName .. ':' .. path)
			redis.call('DEL', 'modified:' .. bucketName .. ':' .. hash)
		`)
	} else {
		function = redis.NewScript(`
			local bucketName = KEYS[1]
			local hash = KEYS[2]
			local path = KEYS[3]
			redis.call('DECR', 'ref_count:' .. bucketName .. ':' .. hash)
			redis.call('DEL', 'ref_file:' .. bucketName .. ':' .. path)
			redis.call('DEL', 'modified:' .. bucketName .. ':' .. hash)
		`)
	}
	return function.Run(redisContext, rdb, []string{bucketName, hash, path}).Err()
}
