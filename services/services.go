package services

import (
	"github.com/gin-gonic/gin"
	"github.com/uopensail/ulib/prome"
	"github.com/uopensail/ulib/utils"
	etcdclient "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type Services struct {
	etcdCli *etcdclient.Client
}

func NewServices() *Services {
	srv := Services{}

	return &srv
}
func (srv *Services) Init(etcdName string, etcdCli *etcdclient.Client, reg utils.Register) {
	srv.etcdCli = etcdCli
	utils.NewMetuxJobUtil(etcdName, reg, etcdCli, 10, -1)

}
func (srv *Services) RegisterGrpc(grpcS *grpc.Server) {

}

func (srv *Services) RegisterGinRouter(ginEngine *gin.Engine) {
	apiV1 := ginEngine.Group("api/v1")
	{
		apiV1.POST("/hello", srv.HelloHandler)
	}

}

func (srv *Services) HelloHandler(c *gin.Context) {
	stat := prome.NewStat("App.HelloHandler")
	defer stat.End()

	c.String(200, "hello world")
	return
}
func (srv *Services) Close() {

}
