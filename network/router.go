package network

import (
	"fmt"
	"sync"

	"github.com/iost-official/Go-IOS-Protocol/core/message"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	sendBlockCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "send_block_count",
			Help: "Count of send block by current node",
		},
	)

	sendTransactionCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "send_transaction_count",
			Help: "Count of send transaction by current node",
		},
	)

	receivedBroadTransactionCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "received_broad_transaction_count",
			Help: "Count of received broad transaction by current node",
		},
	)
)

func init() {
	prometheus.MustRegister(sendBlockCount)
	prometheus.MustRegister(sendTransactionCount)
	prometheus.MustRegister(receivedBroadTransactionCount)
}

// go:generate mockgen -destination mocks/mock_router.go -package protocol_mock github.com/iost-official/Go-IOS-Protocol/network Router

// ReqType Marked request types using by protocol.
type ReqType int32

// const
const (
	ReqPublishTx     ReqType = iota
	ReqBlockHeight           //The height of the request to block
	RecvBlockHeight          //The height of the receiving block
	ReqNewBlock              // recieve a new block or a response for download block
	ReqDownloadBlock         // request for the height of block is equal to target
	BlockHashQuery
	BlockHashResponse
	ReqSyncBlock

	MsgMaxTTL = 2
)

// Router sends specific request and response to other node.
type Router interface {
	Init(base Network, port uint16) error
	FilteredChan(filter Filter) (chan message.Message, error)
	Run()
	Stop()
	Send(req message.Message)
	Broadcast(req message.Message)
	Download(start, end uint64) error
	CancelDownload(start, end uint64) error
	AskABlock(height uint64, to string) error
	QueryBlockHash(start uint64, end uint64) error
}

// Route is a global Router instance.
var Route Router
var once sync.Once

// GetInstance returns singleton of Router.
func GetInstance(conf *NetConfig, target string, port uint16) (Router, error) {
	var err error
	once.Do(func() {
		baseNet, er := NewBaseNetwork(conf)
		if er != nil {
			err = er
			return
		}
		if target == "" {
			target = "base"
		}
		Route, err = RouterFactory(target)
		if err != nil {
			return
		}
		err = Route.Init(baseNet, port)
		if err != nil {
			return
		}
		Route.Run()
	})
	return Route, err
}

// RouterFactory returns different Router instance by target argument.
func RouterFactory(target string) (Router, error) {
	switch target {
	case "base":
		return &RouterImpl{}, nil
	}
	return nil, fmt.Errorf("target Router not found")
}

// RouterImpl is a Router's implement.
type RouterImpl struct {
	base Network

	chIn  <-chan message.Message
	chOut chan<- message.Message

	filterList  []Filter
	filterMap   map[int]chan message.Message
	knownMember []string
	ExitSignal  chan bool

	port uint16
}

// Init inits a RouterImpl.
func (r *RouterImpl) Init(base Network, port uint16) error {
	var err error
	r.base = base
	r.filterList = make([]Filter, 0)
	r.filterMap = make(map[int]chan message.Message)
	r.knownMember = make([]string, 0)
	r.ExitSignal = make(chan bool)
	r.port = port
	r.chIn, err = r.base.Listen(port)
	if err != nil {
		return err
	}
	return nil
}

// FilteredChan returns a filtered request channel.
func (r *RouterImpl) FilteredChan(filter Filter) (chan message.Message, error) {
	chReq := make(chan message.Message, 10000)

	r.filterList = append(r.filterList, filter)
	r.filterMap[len(r.filterList)-1] = chReq

	return chReq, nil
}

func (r *RouterImpl) receiveLoop() {
	for true {
		select {
		case <-r.ExitSignal:
			r.base.Close(r.port)
			return
		case req := <-r.chIn:
			for i, f := range r.filterList {
				if f.check(req) {
					r.filterMap[i] <- req
				}
			}
		}
	}
}

// Run runs a router.
func (r *RouterImpl) Run() {
	go r.receiveLoop()
}

// Stop stops a router.
func (r *RouterImpl) Stop() {
	r.ExitSignal <- true
}

// Send sends a message by router.
func (r *RouterImpl) Send(req message.Message) {
	req.TTL = MsgMaxTTL

	r.base.Send(req)
}

// Broadcast to all known members.
func (r *RouterImpl) Broadcast(req message.Message) {
	req.TTL = MsgMaxTTL

	r.base.Broadcast(req)
}

// LocalID returns local node's ID.
func (r *RouterImpl) LocalID() string {
	return r.base.(*BaseNetwork).localNode.Addr()
}

// Download downloads blocks whose height is greater than start argument and less than end argument.
func (r *RouterImpl) Download(start uint64, end uint64) error {
	fmt.Println("sync:", start, end)
	if end < start {
		return fmt.Errorf("end should be greater than start")
	}
	return r.base.Download(start, end)
}

// CancelDownload cancels downloading blocks.
func (r *RouterImpl) CancelDownload(start uint64, end uint64) error {
	return r.base.CancelDownload(start, end)
}

// AskABlock asks a node for a block.
func (r *RouterImpl) AskABlock(height uint64, to string) error {
	return r.base.AskABlock(height, to)
}

// QueryBlockHash queries blocks' hash by broadcast.
func (r *RouterImpl) QueryBlockHash(start uint64, end uint64) error {
	return r.base.QueryBlockHash(start, end)
}

//Filter is filter used by Router.
// Rulers :
//     1. if both white list and black list are nil, this filter is all-pass
//     2. if one of those is not nil, filter as it is
//     3. if both of those list are not nil, filter as white list
type Filter struct {
	WhiteList  []message.Message
	BlackList  []message.Message
	RejectType []ReqType
	AcceptType []ReqType
}

func (f *Filter) check(req message.Message) bool {
	var memberCheck, typeCheck byte
	if f.WhiteList == nil && f.BlackList == nil {
		memberCheck = byte(0)
	} else if f.WhiteList != nil {
		memberCheck = byte(1)
	} else {
		memberCheck = byte(2)
	}
	if f.AcceptType == nil && f.RejectType == nil {
		typeCheck = byte(0)
	} else if f.AcceptType != nil {
		typeCheck = byte(1)
	} else {
		typeCheck = byte(2)
	}

	var m, t bool

	switch memberCheck {
	case 0:
		m = true
	case 1:
		m = memberContain(req.From, f.WhiteList)
	case 2:
		m = !memberContain(req.From, f.BlackList)
	}

	switch typeCheck {
	case 0:
		t = true
	case 1:
		t = reqTypeContain(req.ReqType, f.AcceptType)
	case 2:
		t = !reqTypeContain(req.ReqType, f.RejectType)
	}

	return m && t
}

func memberContain(a string, c []message.Message) bool {
	for _, m := range c {
		if m.From == a {
			return true
		}
	}
	return false
}

func reqTypeContain(a int32, c []ReqType) bool {
	for _, t := range c {
		if int32(t) == a {
			return true
		}
	}
	return false

}
