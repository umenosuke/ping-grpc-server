package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config a
type Config struct {
	ListenIPAddress       string      `json:"ListenIPAddress"`
	EnableAccessLog       bool        `json:"EnableAccessLog"`
	LoggingPath           tLog        `json:"LoggingPath"`
	UseTLS                bool        `json:"UseTLS"`
	CACertificatePath     string      `json:"CACertificatePath"`
	ServerCertificatePath string      `json:"ServerCertificatePath"`
	ServerPrivateKeyPath  string      `json:"ServerPrivateKeyPath"`
	ICMPSourceIPAddress   string      `json:"ICMPSourceIPAddress"`
	Limit                 tValueLimit `json:"Limit"`
	GrpcStreamBuffer      uint        `json:"BufferGrpcStream"`
}

type tValueLimit struct {
	StopPingerSec         tValueRange `json:"StopPingerSec"`
	IntervalMillisec      tValueRange `json:"IntervalMillisec"`
	TimeoutMillisec       tValueRange `json:"TimeoutMillisec"`
	StatisticsCountsNum   tValueRange `json:"StatisticsCountsNum"`
	StatisticsIntervalSec tValueRange `json:"StatisticsIntervalSec"`
}

type tLog struct {
	Aceess string `json:"Aceess"`
	Error  string `json:"Error"`
}

type tValueRange struct {
	Min uint64 `json:"Min"`
	Max uint64 `json:"Max"`
}

// DefaultConfig a
func DefaultConfig() Config {
	return Config{
		ListenIPAddress: "127.0.0.1:5555",
		EnableAccessLog: true,
		LoggingPath: tLog{
			Aceess: "",
			Error:  "",
		},
		UseTLS:                true,
		CACertificatePath:     "ca.crt",
		ServerCertificatePath: "server.crt",
		ServerPrivateKeyPath:  "server.pem",
		ICMPSourceIPAddress:   "0.0.0.0",
		Limit: tValueLimit{
			StopPingerSec: tValueRange{
				Min: 0,
				Max: 24 * 3600,
			},
			IntervalMillisec: tValueRange{
				Min: 200,
				Max: 10 * 60 * 1000,
			},
			TimeoutMillisec: tValueRange{
				Min: 100,
				Max: 60 * 1000,
			},
			StatisticsCountsNum: tValueRange{
				Min: 1,
				Max: 10000,
			},
			StatisticsIntervalSec: tValueRange{
				Min: 1,
				Max: 3600,
			},
		},
		GrpcStreamBuffer: 5,
	}
}

func crump(value uint64, limit tValueRange) uint64 {
	if value <= limit.Min {
		return limit.Min
	}

	if value >= limit.Max {
		return limit.Max
	}

	return value
}

func configLoad(path string) (Config, error) {
	res := DefaultConfig()

	jsonString, err := ioutil.ReadFile(path)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(jsonString, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func configStringify(data Config) string {
	jsonBlob, _ := json.MarshalIndent(data, "", "")

	return string(jsonBlob)
}
