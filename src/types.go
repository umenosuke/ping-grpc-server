package main

import (
	pb "github.com/umenosuke/ping-grpc-server/proto/pingGrpc"
)

type tStartReq struct {
	id                    uint16
	description           string
	targets               []*pb.StartRequest_IcmpTarget
	intervalMillisec      uint64
	timeoutMillisec       uint64
	stopPingerSec         uint64
	statisticsCountsNum   uint64
	statisticsIntervalSec uint64
}
