# icmp ping を撃ってくれるサーバー

## これは何

- pingを撃つデーモン
- ping対象の指定などはgRPCクライアントから
- 一回のリクエスト内の対象へはpingを並列で撃つ

ping結果を見る箇所と撃ち箇所を別にできるので<br>
PCからのpingが遮断されているときの疎通確認や<br>
AS内部と外部からの疎通確認とかに

## Demo

[クライアント側](https://github.com/umenosuke/ping-grpc-client)を見てください

## 使い方

### 例

実行
```
./ping-grpc-server -config '{"UseTLS":false}'
```


or 設定ファイルを作成
```
./ping-grpc-server -printConfig >> ping-grpc.conf.json
```

ping-grpc.conf.json (設定ファイル)を編集
| 項目 | 意味 | 値 |
|-|-|-|
| ListenIPAddress | gRPCで待ち受けるアドレス | string \`IP\`:\`port\` |
| UseTLS | TLSを利用するか | bool |
|(TLSを利用する場合)|||
| CACertificatePath | CA証明書のパス | string \`file path\` |
| ServerCertificatePath | サーバー証明書のパス | string \`file path\` |
| ServerPrivateKeyPath | サーバー秘密鍵のパス | string \`file path\` |

実行
```
./ping-grpc-server -configPath ping-grpc.conf.json
```

### TLS利用する場合

[ここ](https://github.com/umenosuke/x509helper)などを参考に

- CAの証明書
- サーバー証明書と秘密鍵

を作成してください

### オプションなど

```
$ ./ping-grpc-server -help
Usage of ./ping-grpc-server:
  -config string
        config json string (default "{}")
  -configPath string
        config file path
  -debug
        print debug log
  -printConfig
        show default config
  -version
        show version
```

### コンフィグの内容について
[ここ](https://github.com/umenosuke/ping-grpc-server/blob/master/src/config.go)の
```
type Config struct
```
がそのままエンコードされた形です<br>
値の詳細についてはコメントを参照してください

引数 > 設定ファイル > デフォルト値<br>
の優先度で反映されます

## ビルド方法

### ビルドに必要なもの

- git
- Dockerとか

### コマンド

クローン
```
git clone --recursive git@github.com:umenosuke/ping-grpc-server.git
cd ping-grpc-server
```

ビルド用のコンテナを立ち上げ
```
_USER="$(id -u):$(id -g)" docker-compose -f .docker/docker-compose.yml up -d
```

protoのコンパイル
```
docker exec -it proto_build_ping-grpc-server target_data/.script/proto_build.sh
```

linux&amd64用バイナリを作成(ビルドターゲットは任意で変更してください)<br>
ICMPでraw socketを利用するのでケイパビリティを設定(コマンドをroot権限で実行でも一応大丈夫ですが)
```
docker exec -it go_build_ping-grpc-server target_data/ping-grpc-server/.script/go_build.sh 'linux' 'amd64' 'build/ping-grpc-server'
sudo setcap cap_net_raw=ep 'build/ping-grpc-server'
```

ビルド用のコンテナをお片付け
```
_USER="$(id -u):$(id -g)" docker-compose -f .docker/docker-compose.yml down
```

バイナリはこれ

```
build/ping-grpc-server
```
