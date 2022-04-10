package chain

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gwaylib/errors"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
	RPC_SERVICE_NAME = "RPCService"
)

// TODO: auth score
type RpcService struct {
	host host.Host

	ts    Tipset
	mpool *Mpool

	topicsLK sync.Mutex
	topics   map[string]pubsub.Topic
}

func NewRpcService() *RpcService {
	return &RpcService{
		ts: Tipset{},
		mpool: &Mpool{
			cidPool:  map[string]*types.SignedMessage{},
			addrPool: map[string]*types.SignedMessage{},
		},
		topics: map[string]pubsub.Topic{},
	}
}

var rpcSrv = NewRpcService()

func RpcSrvInstance() *RpcService {
	return rpcSrv
}

func InitRpcSrv(host host.Host) *RpcService {
	// TODO: make a new one?
	rpcSrv.host = host
	return rpcSrv
}

func (r *RpcService) convertTopic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	r.topicsLK.Lock()
	defer r.topicsLK.Unlock()
	cacheTopic, ok := r.topics[topic]
	if ok {
		return &cacheTopic, nil
	}
	if r.host == nil {
		return nil, errors.New("host not init")
	}
	ps, err := pubsub.NewGossipSub(ctx, r.host)
	if err != nil {
		return nil, errors.As(err)
	}

	p2pTopic, err := ps.Join(topic)
	if err != nil {
		return nil, errors.As(err)
	}

	r.topics[topic] = *p2pTopic
	return p2pTopic, nil
}

type PeersArg struct{}
type PeersRet struct {
	Peers peer.IDSlice
}

func (r *RpcService) Peers(ctx context.Context, arg *PeersArg, ret *PeersRet) error {
	ret.Peers = r.host.Peerstore().Peers()
	return nil
}

type ConnectArg struct {
	Addr peer.AddrInfo
}
type ConnectRet struct{}

func (r *RpcService) Connect(ctx context.Context, arg *ConnectArg, ret *ConnectRet) error {
	if err := r.host.Connect(ctx, arg.Addr); err != nil {
		return errors.As(err)
	}
	return nil
}

type PublishArg struct {
	Topic     string
	SignedMsg *types.SignedMessage
}
type PublishRet struct{}

func (r *RpcService) Publish(ctx context.Context, arg *PublishArg, ret *PublishRet) error {
	//sendTitle := build.MessagesTopic(netName)
	topic, err := r.convertTopic(ctx, arg.Topic)
	if err != nil {
		return errors.As(err)
	}
	if err := Publish(ctx, topic, arg.SignedMsg); err != nil {
		return errors.As(err)
	}
	return nil
}

type CurrentTipsetArg struct{}
type CurrentTipsetRet struct {
	Tipset   Tipset
	MpoolLen int
}

func (r *RpcService) CurrentTipset(ctx context.Context, arg *CurrentTipsetArg, ret *CurrentTipsetRet) error {
	ret.Tipset = r.ts
	ret.MpoolLen = r.mpool.Len()
	return nil
}

type CurrentHeightArg struct{}
type CurrentHeightRet struct {
	Height int64
}

func (r *RpcService) CurrentHeight(ctx context.Context, arg *CurrentHeightArg, ret *CurrentHeightRet) error {
	height, err := r.ts.Height()
	if err != nil {
		return errors.As(err)
	}
	ret.Height = height
	return errors.As(err)
}

type CurrentGasInfoArg struct{}
type CurrentGasInfoRet struct {
	ParentBaseFee abi.TokenAmount

	GasInfo
}

func (r *RpcService) CurrentGasInfo(ctx context.Context, arg *CurrentGasInfoArg, ret *CurrentGasInfoRet) error {
	baseFee, err := r.ts.ParentBaseFee()
	if err != nil {
		return errors.As(err)
	}
	ret.ParentBaseFee = baseFee
	ret.GasInfo = r.ts.GasInfo()
	return nil
}