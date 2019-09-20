package main

import (
	"context"
	"crypto/sha1"
	"math/rand"
	"time"

	pb "github.com/fieldkit/app-protocol"
)

func generateModuleId(device *FakeDevice, m *pb.ModuleCapabilities) *pb.ModuleCapabilities {
	hasher := sha1.New()
	hasher.Write([]byte(device.Name))
	hasher.Write([]byte(m.Name))
	moduleID := hasher.Sum(nil)
	m.Id = moduleID
	return m
}

func makeStatusReply(device *FakeDevice) *pb.HttpReply {
	now := time.Now()
	used := uint32(device.State.Streams[0].Size + device.State.Streams[1].Size)
	installed := uint32(512 * 1024 * 1024)

	recording := 0
	if device.State.Recording {
		recording = 1
	}

	return &pb.HttpReply{
		Type: pb.ReplyType_REPLY_STATUS,
		Status: &pb.Status{
			Version:  1,
			Uptime:   1,
			Identity: &device.State.Identity,
			Recording: &pb.Recording{
				Enabled:     uint32(recording),
				StartedTime: device.State.StartedTime,
			},
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
				Path:    "/fk/v1/download/data",
			},
			&pb.DataStream{
				Id:      1,
				Time:    device.State.Streams[1].Time,
				Size:    device.State.Streams[1].Size,
				Version: device.State.Streams[1].Version,
				Block:   device.State.Streams[1].Record,
				Name:    "meta.fkpb",
				Path:    "/fk/v1/download/meta",
			},
		},
		Modules: []*pb.ModuleCapabilities{
			generateModuleId(device, &pb.ModuleCapabilities{
				Position: 0,
				Name:     "Water Quality Module",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "pH",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(device, &pb.ModuleCapabilities{
				Position: 1,
				Name:     "Water Quality Module",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "Dissolved Oxygen",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(device, &pb.ModuleCapabilities{
				Position: 2,
				Name:     "Ocean Module",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "Conductivity",
						UnitOfMeasure: "µS/cm",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        1,
						Name:          "Temperature",
						UnitOfMeasure: "C",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        2,
						Name:          "Depth",
						UnitOfMeasure: "m",
						Frequency:     60,
					},
				},
			}),
		},
	}
}

func handleQueryStatus(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	reply := makeStatusReply(device)
	_, err = rw.WriteReply(reply)
	return
}

func makeLiveReadingsReply(device *FakeDevice) *pb.HttpReply {
	status := makeStatusReply(device)

	now := time.Now()
	ph := rand.Float32() * 7
	conductivity := rand.Float32() * 100
	dissolvedOxygen := rand.Float32() * 10
	temperature := rand.Float32() * 30
	depth := rand.Float32() * 10000

	return &pb.HttpReply{
		Type:    pb.ReplyType_REPLY_READINGS,
		Status:  status.Status,
		Streams: status.Streams,
		Modules: status.Modules,
		LiveReadings: []*pb.LiveReadings{
			&pb.LiveReadings{
				Time: uint64(now.Unix()),
				Modules: []*pb.LiveModuleReadings{
					&pb.LiveModuleReadings{
						Module: status.Modules[0],
						Readings: []*pb.LiveSensorReading{
							&pb.LiveSensorReading{
								Sensor: status.Modules[0].Sensors[0],
								Value:  ph,
							},
						},
					},
					&pb.LiveModuleReadings{
						Module: status.Modules[1],
						Readings: []*pb.LiveSensorReading{
							&pb.LiveSensorReading{
								Sensor: status.Modules[1].Sensors[0],
								Value:  dissolvedOxygen,
							},
						},
					},
					&pb.LiveModuleReadings{
						Module: status.Modules[2],
						Readings: []*pb.LiveSensorReading{
							&pb.LiveSensorReading{
								Sensor: status.Modules[2].Sensors[0],
								Value:  conductivity,
							},
							&pb.LiveSensorReading{
								Sensor: status.Modules[2].Sensors[1],
								Value:  temperature,
							},
							&pb.LiveSensorReading{
								Sensor: status.Modules[2].Sensors[2],
								Value:  depth,
							},
						},
					},
				},
			},
		},
	}
}

func makeBusyReply(delay uint32) *pb.HttpReply {
	return &pb.HttpReply{
		Type: pb.ReplyType_REPLY_BUSY,
		Errors: []*pb.Error{
			&pb.Error{
				Delay: delay,
			},
		},
	}
}

func handleQueryReadings(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if !device.State.ReadingsReady {
		_, err = rw.WriteReply(makeBusyReply(1000))
		return
	}

	reply := makeLiveReadingsReply(device)

	_, err = rw.WriteReply(reply)
	return
}

func handleQueryTakeReadings(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if !device.State.ReadingsReady {
		device.State.ReadingsReady = true
		_, err = rw.WriteReply(makeBusyReply(1000))
		return
	}

	reply := makeLiveReadingsReply(device)

	_, err = rw.WriteReply(reply)
	return
}

func handleConfigure(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if query.Identity.Name != "" {
		device.State.Identity.Device = query.Identity.Name
	}
	reply := makeStatusReply(device)
	_, err = rw.WriteReply(reply)
	return
}

func handleRecordingControl(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if query.Recording.Enabled > 0 {
		device.State.Recording = true
		device.State.StartedTime = uint64(time.Now().Unix())
	} else {
		device.State.Recording = false
		device.State.StartedTime = 0
	}
	reply := makeStatusReply(device)
	_, err = rw.WriteReply(reply)
	return
}
