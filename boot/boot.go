package boot

import (
	"context"
	"fmt"

	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"go.uber.org/zap"

	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
	"github.com/uopensail/ulib/commonconfig"
	"github.com/uopensail/ulib/prome"
	"github.com/uopensail/ulib/utils"
	"github.com/uopensail/ulib/zlog"
	etcdclient "go.etcd.io/etcd/client/v3"
)

type IService interface {
	RegisterGrpc(grpcS *grpc.Server)
	RegisterGinRouter(ginEngine *gin.Engine)
	Init(etcdName string, etcdCli *etcdclient.Client, reg utils.Register)
	Close()
}

var __GITCOMMITINFO__ = ""

// PingPongHandler @Summary 获取标签列表
// @BasePath /
// @Produce  json
// @Success 200 {object} model.StatusResponse
// @Router /ping [get]
func PingPongHandler(gCtx *gin.Context) {
	pStat := prome.NewStat("PingPongHandler")
	defer pStat.End()

	gCtx.JSON(http.StatusOK, struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{
		Code: 0,
		Msg:  "PONG",
	})
	return
}

// @Summary 获取标签列表
// @BasePath /
// @Produce  json
// @Success 200 {object} model.StatusResponse
// @Router /git_hash [get]
func GitHashHandler(gCtx *gin.Context) {
	pStat := prome.NewStat("GitHashHandler")
	defer pStat.End()

	gCtx.String(http.StatusOK, "git_info:"+__GITCOMMITINFO__)
	return
}

type kratosAppRegister struct {
	registry.Registrar
	etcCli       *etcdclient.Client
	regCtxCancel context.CancelFunc
	*kratos.App
}

func (reg *kratosAppRegister) buildInstance() *registry.ServiceInstance {
	instance := registry.ServiceInstance{}
	instance.ID = reg.App.ID()
	instance.Name = reg.App.Name()
	instance.Version = reg.App.Version()
	instance.Endpoints = reg.App.Endpoint()
	instance.Metadata = reg.App.Metadata()
	return &instance
}
func (reg *kratosAppRegister) Register(ctx context.Context) error {
	instance := reg.buildInstance()
	if reg.etcCli != nil {
		cCtx, cancel := context.WithCancel(context.Background())
		reg.Registrar = etcd.New(reg.etcCli, etcd.Context(cCtx))
		reg.regCtxCancel = cancel
	}
	if reg.Registrar != nil {
		return reg.Registrar.Register(ctx, instance)
	}
	return nil
}

func (reg *kratosAppRegister) Deregister(ctx context.Context) error {
	instance := reg.buildInstance()
	if reg.regCtxCancel != nil {
		reg.regCtxCancel()
		reg.regCtxCancel = nil
	}
	if reg.Registrar != nil {
		return reg.Registrar.Deregister(ctx, instance)
	}
	return nil
}

func run(cfg commonconfig.ServerConfig, logDir string, isrv IService) IService {

	zlog.InitLogger(cfg.ProjectName, cfg.Debug, logDir)

	var etcdCli *etcdclient.Client
	if len(cfg.Endpoints) > 0 {
		client, err := etcdclient.New(etcdclient.Config{
			Endpoints: cfg.Endpoints,
		})
		if err != nil {
			zlog.LOG.Fatal("etcd error", zap.Error(err))
		} else {
			etcdCli = client
		}
	}

	options := make([]kratos.Option, 0)

	serverName := cfg.Name
	grpcSrv := newGRPC(cfg.GRPCPort, isrv.RegisterGrpc)
	httpSrv := newHTTPServe(cfg.HTTPPort, isrv.RegisterGinRouter)

	options = append(options, kratos.Name(serverName), kratos.Version(__GITCOMMITINFO__), kratos.Server(
		httpSrv,
		grpcSrv,
	))
	appReg := kratosAppRegister{
		etcCli: etcdCli
	}
	options = append(options, kratos.BeforeStart(func(ctx context.Context) error {
		isrv.Init("microservices/"+serverName, etcdCli, &appReg)
		return nil
	}))

	options = append(options, kratos.AfterStart(func(ctx context.Context) error {
		return appReg.Register(context.Background())
	}))

	app := kratos.New(options...)
	appReg.App = app
	go func() {
		if err := app.Run(); err != nil {
			zlog.LOG.Fatal("run error", zap.Error(err))
		}
	}()

	return isrv
}

func newHTTPServe(httpPort int, registerFunc func(*gin.Engine)) *khttp.Server {
	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())

	url := ginSwagger.URL(fmt.Sprintf("swagger/doc.json")) // The url pointing to API definition
	ginEngine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	ginEngine.GET("/ping", PingPongHandler)
	ginEngine.GET("/git_hash", GitHashHandler)

	registerFunc(ginEngine)
	httpSrv := khttp.NewServer(khttp.Address(fmt.Sprintf(":%d", httpPort)))
	httpSrv.HandlePrefix("/", ginEngine)
	return httpSrv
}

func newGRPC(gRPCPort int, registerFunc func(server *grpc.Server)) *kgrpc.Server {
	grpcSrv := kgrpc.NewServer(
		kgrpc.Address(fmt.Sprintf(":%d",
			gRPCPort)),
		kgrpc.Middleware(
			recovery.Recovery(),
		),
	)
	registerFunc(grpcSrv.Server)
	return grpcSrv
}

func runPProf(port int) {
	if port > 0 {
		go func() {
			fmt.Println(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
		}()
	}
}

func runProme(projectName string, port int) *prome.Exporter {
	//prome的打点
	promeExport := prome.NewExporter(projectName)
	go func() {
		err := promeExport.Start(port)
		if err != nil {
			panic(err)
		}
	}()

	return promeExport
}

func Load(cfg commonconfig.ServerConfig, logDir string, isrv IService) {

	application := run(cfg, logDir, isrv)

	if len(cfg.ProjectName) <= 0 {
		panic("config.ProjectName NULL")
	}

	runPProf(cfg.PProfPort)

	//prome的打点
	promeExport := runProme(cfg.ProjectName, cfg.PromePort)
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " app running....")
	<-signalChanel
	application.Close()
	promeExport.Close()
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " app exit....")
}
