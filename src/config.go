package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config 設定ファイルの中身
type Config struct {
	//gRPCで待ち受けるアドレス(`IP`:`port`)
	ListenIPAddress string `json:"ListenIPAddress"`

	//アクセスログを出力するかどうか
	EnableAccessLog bool `json:"EnableAccessLog"`

	//アクセスログのパス
	LoggingPath tLogPath `json:"LoggingPath"`

	//TLSを利用するかどうか
	UseTLS bool `json:"UseTLS"`

	//CA証明書のパス
	CACertificatePath string `json:"CACertificatePath"`

	//サーバー証明書のパス
	ServerCertificatePath string `json:"ServerCertificatePath"`

	//サーバー秘密鍵のパス
	ServerPrivateKeyPath string `json:"ServerPrivateKeyPath"`

	//ICMPを撃つアドレス(基本0.0.0.0でいいかと)
	ICMPSourceIPAddress string `json:"ICMPSourceIPAddress"`

	//リクエストの値を制限
	Limit tValueLimit `json:"Limit"`

	//gRPCのストリームへ投げる用のチャンネルのバッファ
	GrpcStreamBuffer uint `json:"BufferGrpcStream"`
}

//アクセスログのパス
//空文字列で標準出力へ
type tLogPath struct {
	Aceess string `json:"Aceess"`
	Error  string `json:"Error"`
}

//リクエストの値を制限
type tValueLimit struct {
	//pingを撃ち続ける時間(秒)
	StopPingerSec tValueRange `json:"StopPingerSec"`

	//一つの対象へのpingを撃つインターバル(ミリ秒)
	IntervalMillisec tValueRange `json:"IntervalMillisec"`

	//pingのタイムアウトまでの時間(ミリ秒)
	TimeoutMillisec tValueRange `json:"TimeoutMillisec"`

	//pingの統計をとるため、過去いくつの結果を保持するか
	StatisticsCountsNum tValueRange `json:"StatisticsCountsNum"`

	//pingの統計を集計するインターバル
	StatisticsIntervalSec tValueRange `json:"StatisticsIntervalSec"`
}

//値の下限値と上限値
//Min <= 値 <= MAX
//になるように制限
type tValueRange struct {
	Min uint64 `json:"Min"`
	Max uint64 `json:"Max"`
}

// DefaultConfig is return default value config
func DefaultConfig() Config {
	return Config{
		ListenIPAddress: "127.0.0.1:5555",
		EnableAccessLog: true,
		LoggingPath: tLogPath{
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

func configLoad(configPath string, configJSON string) (Config, error) {
	res := DefaultConfig()

	if configPath != "" {
		jsonString, err := ioutil.ReadFile(configPath)
		if err != nil {
			return res, err
		}
		err = json.Unmarshal(jsonString, &res)
		if err != nil {
			return res, err
		}
	}

	err := json.Unmarshal([]byte(configJSON), &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func configStringify(data Config) string {
	jsonBlob, _ := json.MarshalIndent(data, "", "")

	return string(jsonBlob)
}
