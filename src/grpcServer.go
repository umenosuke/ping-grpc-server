package main

import (
	"context"

	"umenosuke.net/labelinglog"
	pb "umenosuke.net/ping-grpc/proto/go/pingGrpc"
)

type grpcServer struct {
	pingServ *pingerServer
}

// Start a
func (thisServer *grpcServer) Start(ctx context.Context, req *pb.StartRequest) (*pb.PingerID, error) {
	logger.Log(labelinglog.FlgInfo, "Start req : "+req.String())
	return thisServer.pingServ.pingerStartReq(req), nil
}

// Stop a
func (thisServer *grpcServer) Stop(ctx context.Context, id *pb.PingerID) (*pb.Null, error) {
	logger.Log(labelinglog.FlgInfo, "Stop id : "+id.String())

	thisServer.pingServ.pingerStop(uint16(id.GetPingerID()))
	return &pb.Null{}, nil
}

// GetPingerList a
func (thisServer *grpcServer) GetPingerList(ctx context.Context, null *pb.Null) (*pb.PingerList, error) {
	logger.Log(labelinglog.FlgInfo, "GetPingerList")

	return thisServer.pingServ.getPingersIDList(), nil
}

// GetPingerInfo a
func (thisServer *grpcServer) GetPingerInfo(ctx context.Context, id *pb.PingerID) (*pb.PingerInfo, error) {
	logger.Log(labelinglog.FlgInfo, "GetPingerInfo id : "+id.String())

	return thisServer.pingServ.info(uint16(id.GetPingerID())), nil
}

// GetsStatistics a
func (thisServer *grpcServer) GetsStatistics(id *pb.PingerID, server pb.Pinger_GetsStatisticsServer) error {
	logger.Log(labelinglog.FlgInfo, "GetsStatistics id : "+id.String())

	ch := thisServer.pingServ.getsStatistics(uint16(id.GetPingerID()))
	for result := range ch {
		if err := server.Send(result); err != nil {
			return err
		}
	}

	return nil
}

// GetsIcmpResult a
func (thisServer *grpcServer) GetsIcmpResult(id *pb.PingerID, server pb.Pinger_GetsIcmpResultServer) error {
	logger.Log(labelinglog.FlgInfo, "GetsIcmpResult id : "+id.String())

	ch := thisServer.pingServ.getsIcmpResult(uint16(id.GetPingerID()))
	for result := range ch {
		if err := server.Send(result); err != nil {
			return err
		}
	}

	return nil
}
