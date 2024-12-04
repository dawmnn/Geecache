package geecache

import pb "geecache/geecachepb"

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
// PeerPicker 是一个接口，用于根据传入的 key 选择合适的节点（Peer）来处理该请求
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
// PeerGetter 是一个接口，表示远程节点，负责通过 HTTP 从其他节点获取缓存数据
//type PeerGetter interface {
//	Get(group string, key string) ([]byte, error)
//}

// 方法 Get 会发送 HTTP 请求，并从对应的远程节点获取数据。它通过 group 和 key 确定要获取的缓存值。
//返回的 []byte 是缓存的数据，error 则表示请求过程中可能发生的错误。

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
