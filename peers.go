package happycache

import pb "github.com/happyhzr/happycache/happycachepb"

type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

type PeerGetter interface {
	Get(*pb.Request, *pb.Response) error
}
