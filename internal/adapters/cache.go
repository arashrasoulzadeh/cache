package adapters

import "context"

type Cache interface {
	Get(context context.Context, key string) (interface{}, error)
	Set(context context.Context, key string, value interface{}) error
}

type cacheDriver struct {
	Server CacheServer
}

func NewCache(server CacheServer) Cache {
	return &cacheDriver{
		Server: server,
	}
}

func (c *cacheDriver) Get(context context.Context, key string) (interface{}, error) {
	return c.Server.Get(context, key)
}

func (c *cacheDriver) Set(context context.Context, key string, value interface{}) error {
	return c.Server.Set(context, key, value, -1)
}
