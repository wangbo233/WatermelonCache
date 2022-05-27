package GoCache

import pb "GoCache/pb"

// PeerPicker 对应于HTTP服务端
type PeerPicker interface {
	// PickPeer 根据传入的 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 对应于 HTTP 客户端
type PeerGetter interface {
	// Get 用于从对应 group 查找缓存值。
	Get(in *pb.Request, out *pb.Response) error
}
