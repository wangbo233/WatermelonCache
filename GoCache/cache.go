package GoCache

/*
	此文件的主要功能：
		1. 实例化lru，封装get和add方法
		2. 添加互斥锁mutex
*/
import (
	"GoCache/lru"
	"sync"
)

type cache struct {
	mu     sync.Mutex
	lru    *lru.Cache
	nbytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 延迟初始化
	if c.lru == nil {
		c.lru = lru.New(c.nbytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, hit bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}
	if v, hit := c.lru.Get(key); hit {
		return v.(ByteView), hit
	}
	return
}
