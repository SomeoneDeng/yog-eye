package bean

import (
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"
	"yogeye/targetinfo"

	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	Port string `yaml:"port"`
}

type EyeServer struct {
	Conf ServerConfig
}

func (is *EyeServer) TargetInfoReport(ts targetinfo.TargetService_TargetInfoReportServer) error {
	for {

		ti, err := ts.Recv()
		if err != nil {
			log.Println(err.Error())
			break
		}
		log.Println(ti.String())
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
		log.Println("收到心跳 --> ", hearBeat.GetHostKey(), "[", hearBeat.GetBeatTime(), "]")
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

func (is *EyeServer) InitServ(configPath string) {

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
