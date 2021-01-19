package bean

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	"yogeye/targetinfo"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	Port         string `yaml:"port"`
	HTTPPort     string `yaml:"http_port"`
	ClientExpire int    `yaml:"client_expire"`
}

type EyeServer struct {
	Conf ServerConfig

	// StatusOfTargets   target1_name -> [status],
	StatusOfTargets map[string][]targetinfo.TargetInfo

	HeartBeatMap map[string]int64

	HttpServ *gin.Engine
}

func (is *EyeServer) TargetInfoReport(ts targetinfo.TargetService_TargetInfoReportServer) error {
	for {
		ti, err := ts.Recv()
		if err != nil {
			log.Println(err.Error())
			break
		}
		// log.Println("from --> ", ti.HostKey, " --> ", ti.GetCPUpr())

		is.StatusOfTargets[ti.HostKey] = append(is.StatusOfTargets[ti.HostKey], *ti)
		is.StatusOfTargets[ti.HostKey] = is.StatusOfTargets[ti.HostKey][:1]

		ts.Send(&targetinfo.Response{
			Message: strconv.Itoa((int(ti.GetCPUs()) + 1)),
		})
	}

	return nil
}

func (es *EyeServer) TargetHeartBeat(hb targetinfo.TargetService_TargetHeartBeatServer) error {
	for {
		hearBeat, err := hb.Recv()
		if err != nil {
			log.Println(err.Error())
			break
		}
		// log.Println("收到心跳 --> ", hearBeat.GetHostKey(), "[", hearBeat.GetBeatTime(), "]")
		es.HeartBeatMap[hearBeat.GetHostKey()] = hearBeat.GetBeatTime()
		hearBeat.BeatTime = time.Now().Unix()
		hb.Send(hearBeat)
	}
	return nil
}

func (is *EyeServer) readConfig(config string) {
	yamlFile, err := ioutil.ReadFile(config)
	if err != nil {
		log.Println("加载配置文件失败,", err.Error())
		return
	}
	var cnf ServerConfig
	err = yaml.Unmarshal(yamlFile, &cnf)
	if err != nil {
		log.Println("解析配置文件失败,", err.Error())
		return
	}
	is.Conf = cnf
}

func (es *EyeServer) InitHeartBeatManager() {
	es.HeartBeatMap = make(map[string]int64)
	for {
		now := time.Now().Unix()
		for k, v := range es.HeartBeatMap {
			// log.Println(k, v)
			if now-v > int64(es.Conf.ClientExpire) {
				log.Println("client 「", k, "」 expired.")
				delete(es.HeartBeatMap, k)
				delete(es.StatusOfTargets, k)
			}
		}
		// check every 1 sec
		time.Sleep(time.Second * 1)
	}
}

func (is *EyeServer) InitServ(configPath string) {

	go is.InitHttpServ()

	go is.InitHeartBeatManager()

	is.readConfig(configPath)

	//监听端口
	lis, err := net.Listen("tcp", is.Conf.Port)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//创建一个grpc 服务器
	s := grpc.NewServer()
	//注册事件
	targetinfo.RegisterTargetServiceServer(s, is)
	//处理链接
	s.Serve(lis)
}

func (es *EyeServer) InitHttpServ() {
	es.HttpServ = gin.Default()

	yog := es.HttpServ.Group("/yog")
	{
		yog.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, es.StatusOfTargets)
		})

		yog.GET("/hearts", func(c *gin.Context) {
			c.JSON(http.StatusOK, es.HeartBeatMap)
		})
	}

	log.Println("http port --> ", es.Conf.HTTPPort)
	err := es.HttpServ.Run(es.Conf.HTTPPort)
	if err != nil {
		log.Println(err.Error())
	}
}
