package main

import (
	"flag"
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"golang.org/x/sync/errgroup"
	"grayscaleService/common"
	"grayscaleService/repositories"
	"grayscaleService/server"
	httpTransport "grayscaleService/transport/http"
	"grayscaleService/util"
	"net"
	"net/http"
)

var (
	httpAddr   = flag.String("http-addr", ":18183", "HTTP listen address")
	consulAddr = flag.String("consul", "xxx.xx.xxx.4:8500", "consul address")
	minioAddr  = flag.String("minio", "xxx.xx.xxx.xx:9000", "minio address")
)

func main() {
	dbAuth, errDbAuth := common.NewMysqlConn("managePlatform")
	if errDbAuth != nil {
		fmt.Println("error")
		log.Error(errDbAuth)
	}
	// minio连接
	minioClient, errMinio := util.NewMinio(*minioAddr, "minio用户名", "minio密码", false)
	if errMinio != nil {
		fmt.Println("error")
		log.Error(errDbAuth)
	}
	grayscaleModuleRepo := repositories.NewGrayscaleMansger("module", "version", dbAuth)
	bs := server.NewGrayscaleService(grayscaleModuleRepo, *consulAddr, minioClient)

	var g errgroup.Group
	g.Go(func() error {
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			fmt.Printf("http: net.Listen(tcp, %s) failed, err:%v\n", *httpAddr, err)
			return err
		}
		defer httpListener.Close()
		httpHandler := httpTransport.NewHTTPServer(bs)
		return http.Serve(httpListener, httpHandler)
	})
	if err := g.Wait(); err != nil {
		fmt.Printf("server exit with err:%v\n", err)
	}
}
