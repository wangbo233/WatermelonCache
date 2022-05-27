package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 把[]byte映射为uint32
type Hash func(data []byte) uint32

type Map struct {
	hash Hash
	// 虚拟节点的倍数
	replicas int
	// 哈希环
	keys []int
	// 虚拟节点与真实节点的映射表 hashMap
	hashMap map[int]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add : 添加真实节点
func (m *Map) Add(keys ...string) {
	// key是节点的真是名字
	for _, key := range keys {
		// 创建i个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 获取虚拟节点的hash值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 添加到hash环
			m.keys = append(m.keys, hash)
			// 更新映射表
			m.hashMap[hash] = key
		}
	}
	// 环上的hash值排序
	sort.Ints(m.keys)
}

// Get : 选择一个节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
