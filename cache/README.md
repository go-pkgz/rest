
## Cache wrapper for web applications. 

The primary goal is to simplify caching of responses. 

Adds guava-style loading cache and support of scopes for partial flushes. 
Provides in-memory `NewMemoryCache` on top of [hashicorp/golang-lru]("https://github.com/hashicorp/golang-lru") and 
defines basic interface for other implementations.

In addition to `Get` and `Flush` methods, memory cache also support limits for a single value size, number of keys and total memory utilization. `PostFlushFn` adds ability to call a function on flush completion.

## Install and update

`go get -u github.com/go-pkgz/rest/cache`

## Usage

```golang
    // create in-memory cache with max keys=50, total (max) size=2000 and max cached size of a record = 200
    lc, err := cache.NewMemoryCache(cache.MaxKeys(50), cacheMaxCacheSize(2000), cache.MaxValSize(200)) 
    if err != nil {
        panic(err)
    }
    ...

    // load cached value for key1. Call func if not cached yet or evicted
    res, err = lc.Get(cache.NewKey("site1").ID("key1").Scopes("scope1"), func() ([]byte, error) {
		return []byte("1234567890"), nil
    }) 
    
    lc.Flush("scope1") // invalidate cache for scope1
```
