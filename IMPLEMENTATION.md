# Implementation of the protocol

This implementation requires the use of Redis as a database for storing the metadata about the files.
The S3 server must guarantee strict consistency for the protocol to work correctly. 
This implementation (probably) won't work with highly available Redis deployments.

## Used keys in Redis
- `ref_count:{bucket-name}:{sha256-hash} {val}`: File with the sha256-hash is referenced `val` times.
- `ref_file:{bucket-name}:{path} {sha256-hash}`: File with the path has the sha256-hash.
- `modified:{bucket-name}:{path} {last_modified}`: File with the path has the last_modified version.

## `PUT /files/{path}`

### How the proxy handles this request:
1. Check if the server has newer version of the file than the one being uploaded (`GET modified:{bucket-name}:{path}`).
If it does, do nothing.
2. Check if ref-count to the file is 0. If it is, it means that this file is during the process of being deleted -- 
wait for the ref-count to be deleted.
LUA script for checking if the proxy can upload the file:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local ref_count = redis.call('GET', 'ref_count:' .. bucketName .. ':' .. hash) -- Check if the file exists
if ref_count == nil then  -- The file does not exist, set ref_count to 0 to indicate that the file is being processed.
    redis.call('SET', 'ref_count:' .. bucketName .. ':' .. hash, 0, 'EX', 60) -- Set the timeout in case the client crashes.
    return 1
end
if ref_count == 0 then
    return 0
end
return 2
```
Running: `EVAL <script> 2 {bucket-name} {sha256-hash}`
If the result is 0, wait and go to step 1.

3. If the file does not exist on s3 (the script returned `1`), upload the file to s3.
4. Add the file to db atomically:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local path = KEYS[3]
local last_modified = ARGV[1]
redis.call('INCR', 'ref_count:' .. bucketName .. ':' .. hash)
redis.call('SET', 'ref_file:' .. bucketName .. ':' .. path, hash)
redis.call('SET', 'modified:' .. bucketName .. ':' .. path, last_modified)
```
Running: `EVAL <script> 3 {bucket-name} {sha256-hash} {path} {last_modified}`.

##  `DELETE /files/{path}`

### How the proxy handles this request:
1. Check if the file exists in the db (`EXISTS ref_file:{bucket-name}:{path}`) 
and it is uploaded (`GET ref_count:{bucket-name}:{sha256-hash}` should return more than zero). If it doesn't, return 404.
2. Check if the file has newer version (`GET modified:{bucket-name}:{path}`). If it does, do nothing.
2. Decrement the ref-count with the script:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local ref_count = redis.call('DECR', 'ref_count:' .. bucketName .. ':' .. hash)
if ref_count == 0 then -- The file is not referenced anymore, set the ref_count to 0 to indicate that the file is being processed.
    redis.call('EXPIRE', 'ref_count:' .. bucketName .. ':' .. hash, 60) -- Set the timeout in case the client crashes.
    return 1
end
return 0
```
Running: `EVAL <script> 2 {bucket-name} {sha256-hash}`
If the script returns `0`, the file doesn't need deleting, go to step 4. Otherwise go to step 3.

3. Remove the file from s3. Remove the ref-count and the referenced file from the db:
```lua
local bucketName = KEYS[1]
local hash = KEYS[2]
local path = KEYS[3]
redis.call('DEL', 'ref_count:' .. bucketName .. ':' .. hash)
redis.call('DEL', 'ref_file:' .. bucketName .. ':' .. path)
redis.call('DEL', 'modified:' .. bucketName .. ':' .. hash)
```
Running: `EVAL <script> 3 {bucket-name} {sha256-hash} {path}`.
This scripts atomically removes the ref-count and the referenced file from the db. Return with 200 (or sth).
4. Remove the file from the db:
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