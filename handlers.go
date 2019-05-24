package main

import (
	"context"

	pb "github.com/fieldkit/app-protocol"
)

func handleQueryCapabilities(ctx context.Context, wireQuery *pb.WireMessageQuery) (reply *pb.WireMessageReply, err error) {
	reply = &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_CAPABILITIES,
		Capabilities: &pb.Capabilities{
			Version: 0x1,
			Name:    "FieldKit Station",
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

	return
}

func handleQueryStatus(ctx context.Context, wireQuery *pb.WireMessageQuery) (reply *pb.WireMessageReply, err error) {
	reply = &pb.WireMessageReply{
		Type:   pb.ReplyType_REPLY_STATUS,
		Status: &pb.DeviceStatus{},
	}

	return
}

func handleQueryFiles(ctx context.Context, wireQuery *pb.WireMessageQuery) (reply *pb.WireMessageReply, err error) {
	reply = &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_FILES,
		Files: &pb.Files{
			Files: []*pb.File{
				&pb.File{},
				&pb.File{},
				&pb.File{},
				&pb.File{},
			},
		},
	}

	return
}
