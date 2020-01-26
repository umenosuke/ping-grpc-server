package main

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"umenosuke.net/labelinglog"
	pb "umenosuke.net/ping-grpc/proto/go/pingGrpc"
	"umenosuke.net/pinger4"
)

type pingerServer struct {
	chStart chan tStartReq
	pingers *tPingers
	config  Config
}

func newPingerServer(config Config) pingerServer {
	return pingerServer{
		chStart: make(chan tStartReq, 10),
		pingers: &tPingers{list: make(map[uint16]*tPingersEntry)},
		config:  config,
	}
}

func (thisServer *pingerServer) serv(ctx context.Context) {
	wgChild := sync.WaitGroup{}

	(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case request := <-thisServer.chStart:
				wgChild.Add(1)
				go (func() {
					defer wgChild.Done()
					childCtx, childCtxCancel := context.WithCancel(ctx)
					defer childCtxCancel()

					thisServer.pingerStart(childCtx, request)
				})()
			}
		}
	})()

	wgChild.Wait()
}

func (thisServer *pingerServer) pingerStart(ctx context.Context, request tStartReq) {
	wgChild := sync.WaitGroup{}

	config := pinger4.DefaultConfig()
	config.DebugEnable = *debugFlag
	config.SourceIPAddress = thisServer.config.ICMPSourceIPAddress
	limit := thisServer.config.Limit
	config.IntervalMillisec = int64(crump(request.intervalMillisec, limit.IntervalMillisec))
	config.TimeoutMillisec = int64(crump(request.timeoutMillisec, limit.TimeoutMillisec))
	config.StatisticsCountsNum = int64(crump(request.statisticsCountsNum, limit.StatisticsCountsNum))

	id := request.id
	pinger := pinger4.New(int(id), config)
	pinger.SetLogWriter(labelinglog.FlgsetAll, serverLogWriter)
	if *debugFlag {
		pinger.SetLogEnableLevel(labelinglog.FlgsetAll)
	} else {
		pinger.SetLogEnableLevel(labelinglog.FlgsetCommon)
	}

	targets := request.targets
	for _, target := range targets {
		pinger.AddTarget(target.GetTargetIP(), target.GetComment())
	}

	childCtx, childCtxCancel := context.WithCancel(ctx)
	defer childCtxCancel()

	pingerStopTime := time.Duration(crump(request.stopPingerSec, limit.StopPingerSec)) * time.Second

	p := &tPingerWrap{
		pinger:            &pinger,
		idStr:             strconv.Itoa(pinger.GetIcmpID()),
		description:       request.description,
		startUnixNanosec:  uint64(time.Now().UnixNano()),
		expireUnixNanosec: uint64(time.Now().Add(pingerStopTime).UnixNano()),
		cancelFunc:        childCtxCancel,
		chResultListener: struct {
			sync.Mutex
			list []chan<- *pb.IcmpResult
		}{
			list: make([]chan<- *pb.IcmpResult, 0),
		},
		chStatisticsListener: struct {
			sync.Mutex
			list []chan<- *pb.Statistics
		}{
			list: make([]chan<- *pb.Statistics, 0),
		},
		statisticsInterval: crump(request.statisticsIntervalSec, limit.StatisticsIntervalSec),
	}

	wgChild.Add(1)
	go (func() {
		defer wgChild.Done()
		p.start(childCtx)
	})()

	wgChild.Add(1)
	go (func() {
		defer wgChild.Done()
		logger.Log(labelinglog.FlgDebug, "(id "+p.idStr+")"+" Start time.After")
		defer (func() {
			logger.Log(labelinglog.FlgDebug, "(id "+p.idStr+")"+" finish time.After")
			childCtxCancel()
		})()

		select {
		case <-childCtx.Done():
		case <-time.After(pingerStopTime):
		}
	})()

	(func() {
		thisServer.pingers.Lock()
		defer thisServer.pingers.Unlock()
		if pinger, ok := thisServer.pingers.list[id]; ok {
			pinger.entry = p
			close(pinger.running)
		} else {
			childCtxCancel()
		}
	})()

	wgChild.Wait()
	thisServer.pingers.deletePinger(id)
}

func (thisServer *pingerServer) pingerStartReq(req *pb.StartRequest) *pb.PingerID {
	targets := req.GetTargets()
	if targets == nil {
		return &pb.PingerID{}
	} else if len(targets) <= 0 {
		return &pb.PingerID{}
	}

	logger.Log(labelinglog.FlgDebug, "pingers len: "+strconv.Itoa(len(thisServer.pingers.list)))
	id := uint16(rand.Uint32())
	retryCount := 0
	for {
		if retryCount > 0xffff {
			logger.Log(labelinglog.FlgError, "pingerStart Busy")
			return &pb.PingerID{}
		}

		if _, ok := thisServer.pingers.list[id]; ok {
			id = uint16(rand.Uint32())
			retryCount++
		} else {
			break
		}
	}

	thisServer.pingers.addPinger(id, &tPingersEntry{
		running: make(chan struct{}),
		entry:   nil,
	})

	thisServer.chStart <- tStartReq{
		id:                    id,
		description:           req.GetDescription(),
		targets:               targets,
		intervalMillisec:      req.GetIntervalMillisec(),
		timeoutMillisec:       req.GetTimeoutMillisec(),
		stopPingerSec:         req.GetStopPingerSec(),
		statisticsCountsNum:   req.GetStatisticsCountsNum(),
		statisticsIntervalSec: req.GetStatisticsIntervalSec(),
	}

	return &pb.PingerID{
		PingerID: uint32(id),
	}
}

func (thisServer *pingerServer) pingerStop(id uint16) {
	if pinger, ok := thisServer.pingers.getPinger(id); ok {
		<-pinger.running
		if pinger.entry != nil {
			pinger.entry.cancelFunc()
		}
	}
}

func (thisServer *pingerServer) getPingersIDList() *pb.PingerList {
	thisServer.pingers.Lock()
	defer thisServer.pingers.Unlock()

	pingers := make([]*pb.PingerList_PingerSumally, 0, len(thisServer.pingers.list))
	for key, pinger := range thisServer.pingers.list {
		if pinger.entry != nil {
			pingers = append(pingers, &pb.PingerList_PingerSumally{
				PingerID:          uint32(key),
				Description:       pinger.entry.description,
				StartUnixNanosec:  pinger.entry.startUnixNanosec,
				ExpireUnixNanosec: pinger.entry.expireUnixNanosec,
			})
		}
	}

	return &pb.PingerList{
		Pingers: pingers,
	}
}

func (thisServer *pingerServer) info(id uint16) *pb.PingerInfo {
	if pinger, ok := thisServer.pingers.getPinger(id); ok {
		<-pinger.running
		if pinger.entry != nil {
			info := pinger.entry.pinger.GetInfo()

			targets := make([]*pb.PingerInfo_IcmpTarget, 0)

			for _, id := range info.TargetsOrder {
				if target, ok := info.Targets[id]; ok {
					targets = append(targets, &pb.PingerInfo_IcmpTarget{
						TargetID:    uint32(id),
						TargetIP:    target.IPAddress,
						TargetBinIP: pinger4.BinIPv4Address2String(id),
						Comment:     target.Comment,
					})
				}
			}

			return &pb.PingerInfo{
				Description:           pinger.entry.description,
				Targets:               targets,
				IntervalMillisec:      uint64(info.IntervalMillisec),
				TimeoutMillisec:       uint64(info.TimeoutMillisec),
				StatisticsCountsNum:   uint64(info.StatisticsCountsNum),
				StartUnixNanosec:      pinger.entry.startUnixNanosec,
				ExpireUnixNanosec:     pinger.entry.expireUnixNanosec,
				StatisticsIntervalSec: pinger.entry.statisticsInterval,
			}
		}
	}

	return &pb.PingerInfo{}
}

func (thisServer *pingerServer) getsIcmpResult(id uint16) <-chan *pb.IcmpResult {
	ch := make(chan *pb.IcmpResult, thisServer.config.GrpcStreamBuffer)

	if pinger, ok := thisServer.pingers.getPinger(id); ok {
		<-pinger.running
		if pinger.entry != nil {
			pinger.entry.chResultListener.Lock()
			defer pinger.entry.chResultListener.Unlock()
			pinger.entry.chResultListener.list = append(pinger.entry.chResultListener.list, ch)
		} else {
			close(ch)
		}
	} else {
		close(ch)
	}

	return ch
}

func (thisServer *pingerServer) getsStatistics(id uint16) <-chan *pb.Statistics {
	ch := make(chan *pb.Statistics, thisServer.config.GrpcStreamBuffer)

	if pinger, ok := thisServer.pingers.getPinger(id); ok {
		<-pinger.running
		if pinger.entry != nil {
			pinger.entry.chStatisticsListener.Lock()
			defer pinger.entry.chStatisticsListener.Unlock()
			pinger.entry.chStatisticsListener.list = append(pinger.entry.chStatisticsListener.list, ch)
		} else {
			close(ch)
		}
	} else {
		close(ch)
	}

	return ch
}
