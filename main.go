package main

import (
	"flag"

	"github.com/TianqiuHuang/grpc-client-app/pd/auth"
	"github.com/TianqiuHuang/grpc-client-app/pd/fight"
	auth_middle_ware "github.com/TianqiuHuang/grpc-client-app/pkg/auth"
	"github.com/TianqiuHuang/grpc-client-app/pkg/handler"
	"github.com/TianqiuHuang/grpc-client-app/pkg/middleware/jaegerMiddleware"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

var (
	port           string
	addr           string
	authServerAddr string
	tlsCert        string
	tlsKey         string
)

func init() {
	flag.StringVar(&port, "port", "8000", "listen port")
	flag.StringVar(&addr, "addr", "127.0.0.1:8001", "fight svc addr")
	flag.StringVar(&authServerAddr, "auth-server-addr", "127.0.0.1:6666", "auth svc addr")
	flag.StringVar(&tlsCert, "tls-cert", "", "tls cert")
	flag.StringVar(&tlsKey, "tls-key", "", "tls key")
}

func main() {
	flag.Parse()

	gin.DisableConsoleColor()
	r := gin.Default()
	// gin.SetMode(gin.ReleaseMode)

	// add openTracing middleware
	r.Use(jaegerMiddleware.OpenTracingMiddleware())

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		klog.Fatal(err)
	}
	fightSvcClient := fight.NewFightSvcClient(conn)

	authConn, err := grpc.Dial(authServerAddr, grpc.WithInsecure())
	if err != nil {
		klog.Fatal(err)
	}
	authSvcClient := auth.NewAuthServiceClient(authConn)
	authClient := auth_middle_ware.New(authSvcClient)

	group := r.Group("/", auth_middle_ware.AuthMiddleWare(authClient))
	group.GET("/heros", handler.GetAllHeros(fightSvcClient))
	group.GET("/session", handler.LoadSession(fightSvcClient))
	group.PUT("/session", handler.SelectHero(fightSvcClient))
	group.PUT("/session/fight", handler.Fight(fightSvcClient))
	group.POST("/session/archive", handler.Archive(fightSvcClient))
	group.POST("/session/level", handler.Level(fightSvcClient))
	group.POST("/session/quit", handler.Quit(fightSvcClient))

	go func() {
		if err := handler.InitTop10Client(fightSvcClient); err != nil {
			klog.Warning(err)
		}
	}()

	r.GET("/top10", handler.Top10())

	// Admin rest api
	go func() {
		if err := handler.InitAdminClient(fightSvcClient); err != nil {
			klog.Warning(err)
		}
	}()
	adminGroup := r.Group("/admin", auth_middle_ware.AdminAuthMiddleWare(authClient))
	adminGroup.POST("/hero", handler.CreateHero())
	adminGroup.PUT("/hero", handler.AdjustHero())

	if tlsCert != "" && tlsKey != "" {
		r.RunTLS(":"+port, tlsCert, tlsKey)
	} else {
		r.Run(":" + port)
	}
}
