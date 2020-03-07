package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/umenosuke/labelinglog"

	pb "github.com/umenosuke/ping-grpc-server/proto/go/pingGrpc"
)

const terminateTimeOutSec = 5
const debugPrintIntervalSec = 30

var serverLogWriter = os.Stderr

var exitCode = 0

var logger = labelinglog.New("pinger-grpc", serverLogWriter)

var (
	metaVersion  = "unknown"
	metaRevision = "unknown"
)

var (
	argDebugFlag       = flag.Bool("debug", false, "print debug log")
	argConfigPath      = flag.String("configPath", "./ping-grpc.conf.json", "config file path")
	argShowConfigFlg   = flag.Bool("printConfig", false, "show default config")
	argShowVersionFlag = flag.Bool("version", false, "show version")
)

func init() {
	flag.Parse()

	if *argDebugFlag {
		logger.SetEnableLevel(labelinglog.FlgsetAll)
	} else {
		logger.SetEnableLevel(labelinglog.FlgsetCommon)
	}

	rand.Seed(time.Now().UnixNano())
}

func main() {
	subMain()
	os.Exit(exitCode)
}

func subMain() {
	if *argShowVersionFlag {
		fmt.Fprint(os.Stdout, "Version "+metaVersion+"\n"+"Revision "+metaRevision+"\n")
		return
	}

	if *argShowConfigFlg {
		fmt.Fprint(os.Stdout, configStringify(DefaultConfig())+"\n")
		return
	}

	wgFinish := sync.WaitGroup{}

	childCtx, childCtxCancel := context.WithCancel(context.Background())
	defer childCtxCancel()

	config, err := configLoad(*argConfigPath)
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}
	if *argDebugFlag {
		logger.Log(labelinglog.FlgDebug, "now config")
		logger.LogMultiLines(labelinglog.FlgDebug, configStringify(config))
	}

	grpcServerOptions, err := getGrpcServerOptions(childCtx, &wgFinish, config)
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}
	server := grpc.NewServer(grpcServerOptions...)
	pingServ := newPingerServer(config)

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		defer logger.Log(labelinglog.FlgInfo, "finish pingServer")
		logger.Log(labelinglog.FlgInfo, "start pingServer")

		pingServ.serv(childCtx)
	})()

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		defer logger.Log(labelinglog.FlgInfo, "finish grpcServer.Serve")
		logger.Log(labelinglog.FlgInfo, "start grpcServer.Serve "+config.ListenIPAddress)

		listenPort, err := net.Listen("tcp", config.ListenIPAddress)
		if err != nil {
			logger.Log(labelinglog.FlgFatal, err.Error())
			exitCode = 1
			return
		}
		s := &grpcServer{pingServ: &pingServ}
		pb.RegisterPingerServer(server, s)

		if err := server.Serve(listenPort); err != nil {
			logger.Log(labelinglog.FlgFatal, "\""+err.Error()+"\"")
			exitCode = 1
			return
		}
	})()

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		defer logger.Log(labelinglog.FlgInfo, "finish grpcServer.GracefulStop listener")
		logger.Log(labelinglog.FlgInfo, "start grpcServer.GracefulStop listener")

		select {
		case <-childCtx.Done():
			server.GracefulStop()
			return
		}
	})()

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		defer logger.Log(labelinglog.FlgInfo, "finish syscall listener")
		logger.Log(labelinglog.FlgInfo, "start syscall listener")

		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, os.Interrupt)
		for {
			select {
			case <-childCtx.Done():
				return
			case sig := <-c:
				switch sig {
				case syscall.SIGINT:
					fmt.Println()
					logger.Log(labelinglog.FlgDebug, "request stop, SIGINT")
					childCtxCancel()
					return
				default:
					logger.Log(labelinglog.FlgWarn, fmt.Sprintf("unknown syscall [%v]", sig))
				}
			}
		}
	})()

	logger.Log(labelinglog.FlgNotice, "Server Start "+config.ListenIPAddress)

	{
		<-childCtx.Done()

		c := make(chan struct{})
		go (func() {
			wgFinish.Wait()
			close(c)
		})()

		logger.Log(labelinglog.FlgNotice, "waiting for termination ("+strconv.Itoa(terminateTimeOutSec)+"sec)")
		select {
		case <-c:
			logger.Log(labelinglog.FlgNotice, "terminated successfully")
		case <-time.After(time.Duration(terminateTimeOutSec) * time.Second):
			logger.Log(labelinglog.FlgError, "forced termination")
			exitCode = 1
		}
	}
}

func getGrpcServerOptions(ctx context.Context, wgFinish *sync.WaitGroup, config Config) ([]grpc.ServerOption, error) {
	grpcServerOptions := make([]grpc.ServerOption, 0)

	{
		grpcServerOptions = append(grpcServerOptions, grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    1 * time.Second,
			Timeout: 10 * time.Second,
		}))
	}

	if config.UseTLS {
		cert, err :=
			tls.LoadX509KeyPair(
				config.ServerCertificatePath,
				config.ServerPrivateKeyPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		caCert, err := ioutil.ReadFile(config.CACertificatePath)
		if err != nil {
			return nil, err
		}

		if success := certPool.AppendCertsFromPEM(caCert); !success {
			return nil, errors.New("Failed append ca certs")
		}

		creds := credentials.NewTLS(&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    certPool,
			MinVersion:   tls.VersionTLS12,
		})
		grpcServerOptions = append(grpcServerOptions, grpc.Creds(creds))
	}

	if config.EnableAccessLog {
		var accessWriter io.Writer
		if config.LoggingPath.Aceess == "" {
			accessWriter = os.Stdout
		} else {
			f, err := os.OpenFile(config.LoggingPath.Aceess, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return nil, err
			}

			wgFinish.Add(1)
			go (func() {
				defer wgFinish.Done()
				defer logger.Log(labelinglog.FlgInfo, "close AceessLog "+config.LoggingPath.Aceess)
				logger.Log(labelinglog.FlgInfo, "open AceessLog "+config.LoggingPath.Aceess)
				<-ctx.Done()
				f.Close()
			})()
			accessWriter = f
		}
		AceessLogger := labelinglog.New("pinger-grpc Acess", accessWriter)
		AceessLogger.SetEnableLevel(labelinglog.FlgsetAll)
		AceessLogger.DisableFilename()

		var errorWriter io.Writer
		if config.LoggingPath.Error == "" {
			errorWriter = os.Stdout
		} else {
			f, err := os.OpenFile(config.LoggingPath.Error, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return nil, err
			}

			wgFinish.Add(1)
			go (func() {
				defer wgFinish.Done()
				defer logger.Log(labelinglog.FlgInfo, "close ErrorLog "+config.LoggingPath.Error)
				logger.Log(labelinglog.FlgInfo, "open ErrorLog "+config.LoggingPath.Error)
				<-ctx.Done()
				f.Close()
			})()
			errorWriter = f
		}
		ErrorLogger := labelinglog.New("pinger-grpc Error", errorWriter)
		ErrorLogger.SetEnableLevel(labelinglog.FlgsetAll)
		ErrorLogger.DisableFilename()

		grpcServerOptions = append(grpcServerOptions, grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			clientIP := "unknown"
			if p, ok := peer.FromContext(ctx); ok {
				clientIP = p.Addr.String()
			}

			resp, err := handler(ctx, req)
			if err != nil {
				ErrorLogger.Log(labelinglog.FlgWarn, clientIP+" method \""+info.FullMethod+"\" failed err: \""+err.Error()+"\"")
				AceessLogger.Log(labelinglog.FlgWarn, clientIP+" method \""+info.FullMethod+"\" failed")
			} else {
				AceessLogger.Log(labelinglog.FlgNotice, clientIP+" method \""+info.FullMethod+"\" success")
			}
			return resp, err
		}))

		grpcServerOptions = append(grpcServerOptions, grpc.StreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			clientIP := "unknown"
			if p, ok := peer.FromContext(ss.Context()); ok {
				clientIP = p.Addr.String()
			}

			AceessLogger.Log(labelinglog.FlgNotice, clientIP+" method \""+info.FullMethod+"\" start stream")

			err := handler(srv, ss)
			if err != nil {
				code := status.Code(err)
				if code != codes.Canceled {
					ErrorLogger.Log(labelinglog.FlgWarn, clientIP+" method \""+info.FullMethod+"\" stop stream err: \""+err.Error()+"\"")
					AceessLogger.Log(labelinglog.FlgWarn, clientIP+" method \""+info.FullMethod+"\" finish stream with error")
				} else {
					AceessLogger.Log(labelinglog.FlgNotice, clientIP+" method \""+info.FullMethod+"\" finish stream")
				}
			} else {
				AceessLogger.Log(labelinglog.FlgNotice, clientIP+" method \""+info.FullMethod+"\" finish stream")
			}

			return err
		}))
	}

	return grpcServerOptions, nil
}
