package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/blake2b"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/data-protocol"
)

func generateFakeConfiguration() *pb.SignedRecord {
	cfg := &pb.DataRecord{
		Modules: []*pb.ModuleInfo{
			&pb.ModuleInfo{
				Name:     "random-module-1",
				Header:   &pb.ModuleHeader{},
				Firmware: &pb.Firmware{},
				Sensors: []*pb.SensorInfo{
					&pb.SensorInfo{
						Name:          "sensor-0",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-1",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-2",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-3",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-4",
						UnitOfMeasure: "C",
					},
				},
			},
			&pb.ModuleInfo{
				Name:     "random-module-2",
				Header:   &pb.ModuleHeader{},
				Firmware: &pb.Firmware{},
				Sensors: []*pb.SensorInfo{
					&pb.SensorInfo{
						Name:          "sensor-0",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-1",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-2",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-3",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-4",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-5",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-6",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-7",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-8",
						UnitOfMeasure: "C",
					},
					&pb.SensorInfo{
						Name:          "sensor-9",
						UnitOfMeasure: "C",
					},
				},
			},
		},
	}

	body := proto.NewBuffer(make([]byte, 0))
	body.EncodeMessage(cfg)

	hash := blake2b.Sum256(body.Bytes())

	return &pb.SignedRecord{
		Kind: 1, /* Modules */
		Time: 0,
		Data: body.Bytes(),
		Hash: hash[:],
	}
}

func generateFakeReading(reading uint32) *pb.DataRecord {
	now := time.Now()

	return &pb.DataRecord{
		Readings: &pb.Readings{
			Time:    int64(now.Unix()),
			Reading: reading,
			Flags:   0,
			Location: &pb.DeviceLocation{
				Fix:       1,
				Time:      int64(now.Unix()),
				Longitude: -118.2709223,
				Latitude:  34.0318047,
				Altitude:  rand.Float32(),
			},
			SensorGroups: []*pb.SensorGroup{
				&pb.SensorGroup{
					Module: 0,
					Readings: []*pb.SensorAndValue{
						&pb.SensorAndValue{
							Sensor: 0,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 1,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 2,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 3,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 4,
							Value:  rand.Float32(),
						},
					},
				},
				&pb.SensorGroup{
					Module: 1,
					Readings: []*pb.SensorAndValue{
						&pb.SensorAndValue{
							Sensor: 0,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 1,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 2,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 3,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 4,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 5,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 6,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 7,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 8,
							Value:  rand.Float32(),
						},
						&pb.SensorAndValue{
							Sensor: 9,
							Value:  rand.Float32(),
						},
					},
				},
			},
		},
	}
}
