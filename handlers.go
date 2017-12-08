package main

import (
	pb "github.com/fieldkit/app-protocol"
	"log"
	"math/rand"
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
					Id:            0,
					Name:          "Conductivity",
					UnitOfMeasure: "µS/cm",
					Frequency:     60,
				},
				&pb.SensorCapabilities{
					Id:            1,
					Name:          "Temperature",
					UnitOfMeasure: "C",
					Frequency:     60,
				},
				&pb.SensorCapabilities{
					Id:            2,
					Name:          "Depth",
					UnitOfMeasure: "m",
					Frequency:     60,
				},
				&pb.SensorCapabilities{
					Id:            3,
					Name:          "Hydrophone",
					UnitOfMeasure: "",
					Frequency:     0,
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

func rpcQueryLiveData(rc *rpcContext, wireQuery *pb.WireMessageQuery) error {
	log.Printf("Handling %v", wireQuery.QueryDataSets)

	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_LIVE_DATA_POLL,
		LiveData: &pb.LiveData{
			Samples: []*pb.LiveDataSample{
				&pb.LiveDataSample{
					Sensor: 0,
					Time:   uint64(time.Now().Unix()),
					Value:  float32(rand.NormFloat64()*3000 + 200),
				},
				&pb.LiveDataSample{
					Sensor: 1,
					Time:   uint64(time.Now().Unix()),
					Value:  float32(rand.NormFloat64()*32 + 8),
				},
				&pb.LiveDataSample{
					Sensor: 2,
					Time:   uint64(time.Now().Unix()),
					Value:  float32(rand.NormFloat64()*300 + 50),
				},
				&pb.LiveDataSample{
					Sensor: 3,
					Time:   uint64(time.Now().Unix()),
					Value:  0, // float32(rand.NormFloat64()*5 + 2),
				},
			},
		},
	}
	rc.writeMessage(wireReply)

	return nil
}
