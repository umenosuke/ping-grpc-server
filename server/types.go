package main

import (
	pb "umenosuke.net/ping-grpc/proto/go/pingGrpc"
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
