redis-tools
[![Godoc][GodocSVG]][GodocURL]
===

A set of tools around https://godoc.org/github.com/gomodule/redigo/redis.

Pipelining
----------

You can wrap a `redis.Conn` with `Pipeline`. A `Pipeline` allows multiple concurrent callers, provides a simple callback-based API and executes things asynchronously.

Such a design allows you to share a connection among many callers, and save round trip time and fully utilize Redis performance using a simple API.

Example usage:
```go
func MultiIncr(conn redis.Conn, keys []string) (map[string]int64, error) {
	pipeline := NewPipeline(conn)
	defer pipeline.Close()
	res := make(map[string]int64, len(keys))
	var someErr error
	for _, key := range keys {
		curKey := key
		pipeline.Do(func(data interface{}, err error) {
			num, err := redis.Int64(data, err)
			if err != nil {
				someErr = err
				return
			}
			res[curKey] = num
		}, "INCR", key)
	}
	if err := pipeline.Flush(context.Background()); err != nil {
		someErr = err
	}
	return res, someErr
}
```

Note that if you share a pipeline among many functions, you shouldn't call blocking commands, since this will block other users.


   [GodocSVG]: https://godoc.org/github.com/vincent-163/redis-tools/rtools?status.svg
   [GodocURL]: https://godoc.org/github.com/vincent-163/redis-tools/rtools
