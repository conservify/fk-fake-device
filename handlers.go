package main

import (
	pb "github.com/conservify/fk-app-protocol"
	"log"
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
