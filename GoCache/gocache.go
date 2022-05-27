package GoCache

import (
	pb "GoCache/pb"
	"GoCache/singleflight"
	"fmt"
	"log"
	"sync"
)

// Getter 一个Getter根据key加载一个Value
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 定义函数类型
type GetterFunc func(key string) ([]byte, error)

// Get 实现Getter接口的函数
// 函数类型实现某一个接口，称之为接口型函数。
// 方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

/*
	一个Group可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。
	比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info。
	缓存学生课程的命名为 courses。
*/

type Group struct {
	name string
	// 缓存未命中的时候，获取数据的回调
	getter Getter
	// 并发缓存
	mainCache cache
	// 服务端数据结构
	peers PeerPicker
	// 确保并发请求下，每个key只被取回一次
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		mainCache: cache{nbytes: cacheBytes}, // cache中的lru.Cache采用延迟初始化
		getter:    getter,
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup 使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

/*
	Get： 核心函数
	如果键为空，返回一个错误
	如果命中，返回value
	如果没有命中，加载key对应的value
*/
func (g *Group) Get(key string) (ByteView, error) {
	// 如果键为空，返回一个错误
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, hit := g.mainCache.get(key); hit {
		log.Println("[GoCache] hit")
		return v, nil
	}
	return g.load(key)
}

// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。
// 若是本机节点或失败，则回退到 getLocally()
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用用户回调函数获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 将源数据添加到缓存 mainCache 中
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// RegisterPeers 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
