package main

import (
	"context"
	"math/rand"
	"time"

	pb "github.com/fieldkit/app-protocol"
)

func handleQueryStatus(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	now := time.Now()

	used := uint32(device.State.Streams[0].Size + device.State.Streams[1].Size)
	installed := uint32(512 * 1024 * 1024)

	reply := &pb.HttpReply{
		Type: pb.ReplyType_REPLY_STATUS,
		Status: &pb.Status{
			Version:  1,
			Uptime:   1,
			Identity: &device.State.Identity,
			Memory: &pb.MemoryStatus{
				SramAvailable:           128 * 1024,
				ProgramFlashAvailable:   600 * 1024,
				ExtendedMemoryAvailable: 0,
				DataMemoryInstalled:     installed,
				DataMemoryUsed:          used,
				DataMemoryConsumption:   float32(used) / float32(installed) * 100.0,
			},
			Gps: &pb.GpsStatus{
				Fix:        1,
				Time:       uint64(now.Unix()),
				Satellites: 5,
				Longitude:  -118.2709223,
				Latitude:   34.0318047,
				Altitude:   rand.Float32(),
			},
			Power: &pb.PowerStatus{
				Battery: &pb.BatteryStatus{
					Voltage:    3420.0,
					Percentage: 70.0,
				},
			},
		},
		Streams: []*pb.DataStream{
			&pb.DataStream{
				Id:      0,
				Time:    device.State.Streams[0].Time,
				Size:    device.State.Streams[0].Size,
				Version: device.State.Streams[0].Version,
				Block:   device.State.Streams[0].Record,
				Name:    "data.fkpb",
				Path:    "/fk/v1/download/0",
			},
			&pb.DataStream{
				Id:      1,
				Time:    device.State.Streams[1].Time,
				Size:    device.State.Streams[1].Size,
				Version: device.State.Streams[1].Version,
				Block:   device.State.Streams[1].Record,
				Name:    "meta.fkpb",
				Path:    "/fk/v1/download/1",
			},
		},
		Modules: []*pb.ModuleCapabilities{
			&pb.ModuleCapabilities{
				Id:   0,
				Name: "Water Quality Module",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Id:            0,
						Name:          "pH",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			},
			&pb.ModuleCapabilities{
				Id:   1,
				Name: "Water Quality Module",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Id:            0,
						Name:          "Dissolved Oxygen",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			},
			&pb.ModuleCapabilities{
				Id:   2,
				Name: "Ocean Module",
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
				},
			},
		},
	}

	_, err = rw.WriteReply(reply)
	return
}

func handleQueryReadings(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	now := time.Now()

	ph := rand.Float32() * 7
	conductivity := rand.Float32() * 100
	dissolvedOxygen := rand.Float32() * 10
	temperature := rand.Float32() * 30
	depth := rand.Float32() * 10000

	reply := &pb.HttpReply{
		Type: pb.ReplyType_REPLY_READINGS,
		Readings: []*pb.Readings{
			&pb.Readings{
				Time: uint64(now.Unix()),
				Modules: []*pb.ModuleReadings{
					&pb.ModuleReadings{
						Module: 0,
						Readings: []*pb.SensorAndValue{
							&pb.SensorAndValue{
								Sensor: 0,
								Value:  ph,
							},
						},
					},
					&pb.ModuleReadings{
						Module: 1,
						Readings: []*pb.SensorAndValue{
							&pb.SensorAndValue{
								Sensor: 0,
								Value:  dissolvedOxygen,
							},
						},
					},
					&pb.ModuleReadings{
						Module: 2,
						Readings: []*pb.SensorAndValue{
							&pb.SensorAndValue{
								Sensor: 0,
								Value:  conductivity,
							},
							&pb.SensorAndValue{
								Sensor: 1,
								Value:  temperature,
							},
							&pb.SensorAndValue{
								Sensor: 2,
								Value:  depth,
							},
						},
					},
				},
			},
		},
	}

	_, err = rw.WriteReply(reply)
	return
}
