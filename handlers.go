package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/drhodes/golorem"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
	pbatlas "github.com/fieldkit/atlas-protocol"
)

func generateAtlasStatus(device *FakeDevice, index int, delimitted bool) []byte {
	module := device.Modules[index]
	value := module.Calibration

	reply := &pbatlas.WireAtlasReply{
		Type: pbatlas.ReplyType_REPLY_STATUS,
		Calibration: &pbatlas.AtlasCalibrationStatus{
			Type: module.SensorType,
			Raw:  value,
		},
	}

	switch module.SensorType {
	case pbatlas.SensorType_SENSOR_PH:
		reply.Calibration.Ph = pbatlas.PhCalibrations(value)
	case pbatlas.SensorType_SENSOR_DO:
		reply.Calibration.DissolvedOxygen = pbatlas.DoCalibrations(value)
	case pbatlas.SensorType_SENSOR_TEMP:
		reply.Calibration.Temp = pbatlas.TempCalibrations(value)
	case pbatlas.SensorType_SENSOR_ORP:
		reply.Calibration.Orp = pbatlas.OrpCalibrations(value)
	case pbatlas.SensorType_SENSOR_EC:
		reply.Calibration.Ec = pbatlas.EcCalibrations(value)
	}

	data, err := proto.Marshal(reply)
	if err != nil {
		panic(err)
	}

	if delimitted {
		buf := proto.NewBuffer(make([]byte, 0))
		buf.EncodeRawBytes(data)
		return buf.Bytes()
	}

	return data
}

func generateModuleId(position int, device *FakeDevice, m *pb.ModuleCapabilities) *pb.ModuleCapabilities {
	hasher := sha1.New()
	hasher.Write([]byte(device.Name))
	hasher.Write([]byte(m.Name))
	hasher.Write([]byte(fmt.Sprintf("%d", position)))
	moduleID := hasher.Sum(nil)
	m.Id = moduleID
	return m
}

func makeModules(device *FakeDevice) []*pb.ModuleCapabilities {
	if len(device.Modules) == 0 {
		return make([]*pb.ModuleCapabilities, 0)
	}
	return []*pb.ModuleCapabilities{
		generateModuleId(0, device, &pb.ModuleCapabilities{
			Position: 0,
			Name:     "modules.water.ph",
			Status:   generateAtlasStatus(device, 0, false),
			Sensors: []*pb.SensorCapabilities{
				&pb.SensorCapabilities{
					Number:        0,
					Name:          "ph",
					UnitOfMeasure: "",
					Frequency:     60,
				},
			},
		}),
		generateModuleId(1, device, &pb.ModuleCapabilities{
			Position: 1,
			Name:     "modules.water.ec",
			Status:   generateAtlasStatus(device, 1, false),
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
			Name:     "modules.water.temp",
			Status:   generateAtlasStatus(device, 2, false),
			Sensors: []*pb.SensorCapabilities{
				&pb.SensorCapabilities{
					Number:        0,
					Name:          "temp",
					UnitOfMeasure: "",
					Frequency:     60,
				},
			},
		}),
		generateModuleId(3, device, &pb.ModuleCapabilities{
			Position: 3,
			Name:     "modules.water.do",
			Status:   generateAtlasStatus(device, 3, false),
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
			Name:     "modules.water.orp",
			Status:   generateAtlasStatus(device, 4, false),
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
			Name:     "modules.diagnostics",
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
			Name:     "modules.random",
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
	}
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
				Longitude:  device.Longitude,
				Latitude:   device.Latitude,
				Altitude:   rand.Float32(),
			},
			Power: &pb.PowerStatus{
				Battery: &pb.BatteryStatus{
					Voltage:    3420.0,
					Percentage: 70.0,
				},
			},
			Logs: lorem.Paragraph(10, 10),
			Firmware: &pb.Firmware{
				Build:     "build",
				Timestamp: uint64(now.Unix()),
				Version:   "version",
				Hash:      "hash",
				Number:    "590",
			},
		},
		LoraSettings: device.State.Lora,
		NetworkSettings: &pb.NetworkSettings{
			Networks: device.State.Networks,
		},
		Modules: makeModules(device),
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
		Schedules: &pb.Schedules{
			Readings: device.ReadingsSchedule,
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
	}
}

func handleQueryScanNetworks(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	reply := &pb.HttpReply{
		Type: pb.ReplyType_REPLY_SUCCESS,
		NearbyNetworks: &pb.NearbyNetworks{
			Networks: []*pb.NearbyNetwork{
				&pb.NearbyNetwork{
					Ssid: "Cottonwood",
				},
				&pb.NearbyNetwork{
					Ssid: "Conservify",
				},
			},
		},
	}
	_, err = rw.WriteReply(reply)
	return
}

func handleQueryStatus(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if query.Locate != nil {
		device.Latitude = query.Locate.Latitude
		device.Longitude = query.Locate.Longitude
	}
	reply := makeStatusReply(device)
	_, err = rw.WriteReply(reply)
	return
}

func makeDiagnosticsReadings(status *pb.HttpReply) *pb.LiveModuleReadings {
	return &pb.LiveModuleReadings{
		Module: status.Modules[5],
		Readings: []*pb.LiveSensorReading{
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[0],
				Value:  68,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[1],
				Value:  4000,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[2],
				Value:  1024 * 20,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[3],
				Value:  10000,
			},
			&pb.LiveSensorReading{
				Sensor: status.Modules[5].Sensors[4],
				Value:  22,
			},
		},
	}
}

func makeRandomReadings(status *pb.HttpReply) *pb.LiveModuleReadings {
	return &pb.LiveModuleReadings{
		Module: status.Modules[6],
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

	liveReadings := make([]*pb.LiveModuleReadings, 0)

	if len(device.Modules) > 0 {
		liveReadings = []*pb.LiveModuleReadings{
			makeWaterReadings(status, 0), // ph
			makeWaterReadings(status, 1), // ec
			makeWaterReadings(status, 2), // temp
			makeWaterReadings(status, 3), // do
			makeWaterReadings(status, 4), // orp
			makeDiagnosticsReadings(status),
			makeRandomReadings(status),
		}
	}

	return &pb.HttpReply{
		Type:      pb.ReplyType_REPLY_READINGS,
		Status:    status.Status,
		Streams:   status.Streams,
		Modules:   status.Modules,
		Schedules: status.Schedules,
		LiveReadings: []*pb.LiveReadings{
			&pb.LiveReadings{
				Time:    uint64(now.Unix()),
				Modules: liveReadings,
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
	reply := makeLiveReadingsReply(device)

	_, err = rw.WriteReply(reply)
	return
}

func handleQueryTakeReadings(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, rw ReplyWriter) (err error) {
	if query.Locate != nil {
		device.Latitude = query.Locate.Latitude
		device.Longitude = query.Locate.Longitude
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
	if query.Schedules != nil {
		if query.Schedules.Readings != nil {
			if query.Schedules.Readings.Intervals != nil {
				for _, interval := range query.Schedules.Readings.Intervals {
					log.Printf("interval: %v", interval)
				}
			}
			device.ReadingsSchedule = query.Schedules.Readings
			log.Printf("modified schedule: %v", *device.ReadingsSchedule)
		}
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
