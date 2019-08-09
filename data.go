package main

import (
	"math/rand"
	"time"

	pb "github.com/fieldkit/data-protocol"
)

func generateFakeConfiguration() *pb.DataRecord {
	return &pb.DataRecord{}
}

func generateFakeReading(reading uint32) *pb.DataRecord {
	now := time.Now()

	return &pb.DataRecord{
		Readings: &pb.Readings{
			Time:    uint64(now.Unix()),
			Reading: reading,
			Flags:   0,
			Location: &pb.DeviceLocation{
				Fix:       1,
				Time:      uint64(now.Unix()),
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
						&pb.SensorAndValue{
							Sensor: 10,
							Value:  rand.Float32(),
						},
					},
				},
			},
		},
	}
}
