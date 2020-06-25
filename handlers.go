package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"time"

	pb "github.com/fieldkit/app-protocol"
)

func generateModuleId(position int, device *FakeDevice, m *pb.ModuleCapabilities) *pb.ModuleCapabilities {
	hasher := sha1.New()
	hasher.Write([]byte(device.Name))
	hasher.Write([]byte(m.Name))
	hasher.Write([]byte(fmt.Sprintf("%d", position)))
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
				Enabled:     recording > 0,
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
		Schedules: &pb.Schedules{
			Readings: &pb.Schedule{
				Interval: 60,
			},
			Lora: &pb.Schedule{
				Interval: 300,
			},
			Network: &pb.Schedule{
				Interval: 0,
			},
			Gps: &pb.Schedule{
				Interval: 86400,
			},
		},
		LoraSettings: device.State.Lora,
		NetworkSettings: &pb.NetworkSettings{
			Networks: device.State.Networks,
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
			generateModuleId(0, device, &pb.ModuleCapabilities{
				Position: 0,
				Name:     "water",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "ph",
						UnitOfMeasure: "",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "zero",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(1, device, &pb.ModuleCapabilities{
				Position: 1,
				Name:     "water",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "ec",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(2, device, &pb.ModuleCapabilities{
				Position: 2,
				Name:     "water",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "temp",
						UnitOfMeasure: "",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "zero",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(3, device, &pb.ModuleCapabilities{
				Position: 3,
				Name:     "water",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "do",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(4, device, &pb.ModuleCapabilities{
				Position: 4,
				Name:     "water",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "orp",
						UnitOfMeasure: "",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(0xff, device, &pb.ModuleCapabilities{
				Position: 0xff,
				Flags:    1,
				Name:     "diagnostics",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "battery_charge",
						UnitOfMeasure: "%",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        1,
						Name:          "battery_voltage",
						UnitOfMeasure: "mv",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        2,
						Name:          "memory",
						UnitOfMeasure: "bytes",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        3,
						Name:          "uptime",
						UnitOfMeasure: "ms",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        4,
						Name:          "temperature",
						UnitOfMeasure: "C",
						Frequency:     60,
					},
				},
			}),
			generateModuleId(0xff, device, &pb.ModuleCapabilities{
				Position: 0xff,
				Flags:    1,
				Name:     "random",
				Sensors: []*pb.SensorCapabilities{
					&pb.SensorCapabilities{
						Number:        0,
						Name:          "random_0",
						UnitOfMeasure: "",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        1,
						Name:          "random_1",
						UnitOfMeasure: "",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        2,
						Name:          "random_2",
						UnitOfMeasure: "",
						Frequency:     60,
					},
					&pb.SensorCapabilities{
						Number:        3,
						Name:          "random_3",
						UnitOfMeasure: "",
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

func makeDiagnosticsReadings(status *pb.HttpReply) *pb.LiveModuleReadings {
	return &pb.LiveModuleReadings{
		Module: status.Modules[0],
		Readings: []*pb.LiveSensorReading{
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[0],
				Value:  0,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[1],
				Value:  0,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[2],
				Value:  0,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[3],
				Value:  0,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[4],
				Value:  0,
			},
		},
	}
}

func makeRandomReadings(status *pb.HttpReply) *pb.LiveModuleReadings {
	return &pb.LiveModuleReadings{
		Module: status.Modules[0],
		Readings: []*pb.LiveSensorReading{
			&pb.LiveSensorReading{
				Sensor: status.Modules[6].Sensors[0],
				Value:  rand.Float32(),
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[6].Sensors[1],
				Value:  rand.Float32(),
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[6].Sensors[2],
				Value:  rand.Float32(),
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[6].Sensors[3],
				Value:  rand.Float32(),
			},
		},
	}
}

func makeWaterReadings(status *pb.HttpReply, moduleIndex int) *pb.LiveModuleReadings {
	value := float32(7.0) + (rand.Float32()*2 - 1)
	if moduleIndex == 0 {
		return &pb.LiveModuleReadings{
			Module: status.Modules[moduleIndex],
			Readings: []*pb.LiveSensorReading{
				&pb.LiveSensorReading{
					Sensor: status.Modules[moduleIndex].Sensors[0],
					Value:  value,
				},
				&pb.LiveSensorReading{
					Sensor: status.Modules[moduleIndex].Sensors[1],
					Value:  0,
				},
			},
		}
	}
	return &pb.LiveModuleReadings{
		Module: status.Modules[moduleIndex],
		Readings: []*pb.LiveSensorReading{
			&pb.LiveSensorReading{
				Sensor: status.Modules[moduleIndex].Sensors[0],
				Value:  value,
			},
		},
	}
}

func makeLiveReadingsReply(device *FakeDevice) *pb.HttpReply {
	status := makeStatusReply(device)

	now := time.Now()
	// ph := rand.Float32() * 7
	// conductivity := rand.Float32() * 100
	// dissolvedOxygen := rand.Float32() * 10
	// temperature := rand.Float32() * 30
	// depth := rand.Float32() * 10000

	return &pb.HttpReply{
		Type:    pb.ReplyType_REPLY_READINGS,
		Status:  status.Status,
		Streams: status.Streams,
		Modules: status.Modules,
		LiveReadings: []*pb.LiveReadings{
			&pb.LiveReadings{
				Time: uint64(now.Unix()),
				Modules: []*pb.LiveModuleReadings{
					makeDiagnosticsReadings(status),
					makeRandomReadings(status),
					makeWaterReadings(status, 0),
					makeWaterReadings(status, 1),
					makeWaterReadings(status, 2),
					makeWaterReadings(status, 3),
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
	if query.Identity != nil && query.Identity.Name != "" {
		device.State.Identity.Device = query.Identity.Name
	}
	if query.NetworkSettings != nil {
		device.State.Networks = query.NetworkSettings.Networks
	}
	if query.LoraSettings != nil {
		deviceEui := device.State.Lora.DeviceEui
		device.State.Lora = query.LoraSettings
		if device.State.Lora.DeviceEui == nil {
			device.State.Lora.DeviceEui = deviceEui
		}
		device.State.Lora.Modifying = false
	}
	reply := makeStatusReply(device)
	_, err = rw.WriteReply(reply)
	return
}

func handleRecordingControl(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if query.Recording.Enabled {
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
