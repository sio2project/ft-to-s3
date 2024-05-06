# Implementation of the protocol

This implementation requires the use of Redis as a database for storing the metadata about the files.
The S3 server must guarantee strict consistency for the protocol to work correctly. 
This implementation (probably) won't work with highly available Redis deployments.

## Used keys in Redis
- `ref_count:{bucket-name}:{sha256-hash} {val}`: File with the sha256-hash is referenced `val` times.
- `ref_file:{bucket-name}:{path} {sha256-hash}`: File with the path has the sha256-hash.
- `lock:{bucket-name}:{sha256-hash} 1 EX 60 NX`: Lock for the file with the sha256-hash.
Used when deleting, uploading or overwriting the file. Get doesn't have to care about the lock, because S3 should be strictly consistent. 
- `modified:{bucket-name}:{path} {last_modified}`: File with the path has the last_modified version.

## `PUT /files/{path}`

### How the proxy handles this request:
1. Check if the proxy can add the file with the script:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local last_modified = ARGV[1]

local db_modified = redis.call('GET', 'modified:' .. bucketName .. ':' .. hash)
local ref_count = redis.call('GET', 'ref_count:' .. bucketName .. ':' .. hash) -- Check if the file exists
if ref_count == nil or (db_modified != nill and db_modified < last_modified) then
    -- The file does not exist or the file is older than the one being uploaded.
    local ok = redis.call('SET', 'lock:' .. bucketName .. ':' .. hash, 1, 'EX', 60, 'NX') -- Lock the file to prevent other clients from editing it.
    if ok == nil then
        return 0 -- The file is being processed by another client
    end
    return 1 -- The file can be uploaded
end
return 2 -- The file is newer than the one being uploaded
```
Running: `EVAL <script> 2 {bucket-name} {sha256-hash} {last_modified}`. \
If the returned value is `0`, the file is being processed by another client, go to step 1. \
If the returned value is `1`, the file can be uploaded. \
If the returned value is `2`, the existing file is newer than the one being uploaded, exit with 200 (or sth). 

3. Upload the file to s3.
4. Add the file to db atomically:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local path = KEYS[3]
local last_modified = ARGV[1]
redis.call('INCR', 'ref_count:' .. bucketName .. ':' .. hash)
redis.call('SET', 'ref_file:' .. bucketName .. ':' .. path, hash)
redis.call('SET', 'modified:' .. bucketName .. ':' .. path, last_modified)
redis.call('DEL', 'lock:' .. bucketName .. ':' .. hash)
```
Running: `EVAL <script> 3 {bucket-name} {sha256-hash} {path} {last_modified}`.

##  `DELETE /files/{path}`

### How the proxy handles this request:
1. Check if the file exists in the db (`EXISTS ref_file:{bucket-name}:{path}`) 
and it is uploaded (`GET ref_count:{bucket-name}:{sha256-hash}` should return more than zero). If it doesn't, return 404.
2. Check if the file has newer version (`GET modified:{bucket-name}:{path}`). If it does, do nothing.
3. Check if the file can be deleted with the script: 
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
ref_count = redis.call('DECR', 'ref_count:' .. bucketName .. ':' .. hash)
if ref_count == 0 then -- The file is not referenced anymore, set the lock for the file to prevent other clients from processing it.
    ret = redis.Call('SET', 'lock:' .. bucketName .. ':' .. hash, 1, 'EX', 60, 'NX')
    if ret == nil then
        return 0
    end
    return 1
end
return 2
```
Running: `EVAL <script> 2 {bucket-name} {sha256-hash}` \
If the returned value is `0`, the file is being processed by another client, go to step 3. \
If the returned value is `1`, the file can be deleted, go to step 4. \
If the returned value is `2`, the file is still referenced, go to step 5.

4. Remove the file from s3. Remove the ref-count and the referenced file from the db:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local path = KEYS[3]
redis.call('DEL', 'ref_count:' .. bucketName .. ':' .. hash)
redis.call('DEL', 'ref_file:' .. bucketName .. ':' .. path)
redis.call('DEL', 'modified:' .. bucketName .. ':' .. hash)
redis.call('DEL', 'lock:' .. bucketName .. ':' .. hash)
```
Running: `EVAL <script> 3 {bucket-name} {sha256-hash} {path}`.
This scripts atomically removes the lock, ref-count and the referenced file from the db. Return with 200 (or sth).

5. Remove the file from the db:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local path = KEYS[3]
redis.call('DECR', 'ref_count:' .. bucketName .. ':' .. hash)
redis.call('DEL', 'ref_file:' .. bucketName .. ':' .. path)
redis.call('DEL', 'modified:' .. bucketName .. ':' .. hash)
```
Running: `EVAL <script> 3 {bucket-name} {sha256-hash} {path}`.

## `GET /files/{path}`

### How the proxy handles this request:
1. Check if the file exists in the db (`EXISTS ref_file:{bucket-name}:{path}`). If it doesn't, return 404.
2. Get the sha256-hash of the file from the db (`GET ref_file:{bucket-name}:{path}`).
3. Request the file by the sha256-hash from s3. Handle the response.


## File cleaner
The file cleaner is a separate service that runs periodically and removes the files that are not referenced anymore.
It does two things:
- iterates over all `ref_file` keys and checks if the file referenced by the hash still exists in `ref_count`.
If it doesn't, it removes the entry from `ref_file` and `modified`.
- iterates over all files in s3 and checks if the file still exists in the `ref_count`.
If it doesn't, it removes the file from s3 and the entries from `ref_file` and `modified`.

These situations can happen when the client crashes during the file upload or delete.