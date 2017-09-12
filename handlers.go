package main

import (
	pb "github.com/conservify/fk-app-protocol"
	"log"
	"time"
)

func rpcQueryCapabilities(rc *rpcContext, wireQuery *pb.WireMessageQuery) error {
	log.Printf("Handling %v", wireQuery.QueryCapabilities)

	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_CAPABILITIES,
		Capabilities: &pb.Capabilities{
			Version: 0x1,
			Name:    "NOAA-CTD",
			Sensors: []*pb.SensorCapabilities{
				&pb.SensorCapabilities{
					Id:        0,
					Name:      "Conductivity",
					Frequency: 60,
				},
				&pb.SensorCapabilities{
					Id:        1,
					Name:      "Temperature",
					Frequency: 60,
				},
				&pb.SensorCapabilities{
					Id:        2,
					Name:      "Depth",
					Frequency: 60,
				},
				&pb.SensorCapabilities{
					Id:        3,
					Name:      "Hydrophone",
					Frequency: 0,
				},
			},
		},
	}
	rc.writeMessage(wireReply)

	return nil
}

func rpcQueryDataSets(rc *rpcContext, wireQuery *pb.WireMessageQuery) error {
	log.Printf("Handling %v", wireQuery.QueryDataSets)

	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_DATA_SETS,
		DataSets: &pb.DataSets{
			DataSets: []*pb.DataSet{
				&pb.DataSet{
					Id:     0,
					Name:   "Conductivity",
					Sensor: 0,
					Size:   100,
					Time:   uint64(time.Now().Unix()),
					Hash:   0,
				},
				&pb.DataSet{
					Id:     1,
					Name:   "Temperature",
					Sensor: 1,
					Size:   100,
					Time:   uint64(time.Now().Unix()),
					Hash:   0,
				},
				&pb.DataSet{
					Id:     2,
					Name:   "Depth",
					Sensor: 2,
					Size:   100,
					Time:   uint64(time.Now().Unix()),
					Hash:   0,
				},
				&pb.DataSet{
					Id:     3,
					Name:   "Hydrophone",
					Sensor: 3,
					Size:   100,
					Time:   uint64(time.Now().Unix()),
					Hash:   0,
				},
			},
		},
	}
	rc.writeMessage(wireReply)

	return nil
}
