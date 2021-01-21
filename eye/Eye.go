package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"yogeye/targetinfo"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type ClientConf struct {
	ServPort  string `yaml:"serv_port"`
	ClientKey string `yaml:"client_key"`
	AuthKey   string `yaml:"auth"`
}

type Client struct {
	ClientConf ClientConf
}

func (c *Client) readConfig(config string) {
	yamlFile, err := ioutil.ReadFile(config)
	if err != nil {
		log.Println("加载配置文件失败,", err.Error())
		return
	}
	var cnf ClientConf
	err = yaml.Unmarshal(yamlFile, &cnf)
	if err != nil {
		log.Println("解析配置文件失败,", err.Error())
		return
	}
	c.ClientConf = cnf
}

// InitClient init client
func (c *Client) InitClient(config string) {
	c.readConfig(config)
	log.Println("「client start」")
	conn, err := grpc.Dial(c.ClientConf.ServPort, grpc.WithInsecure())
	if err != nil {
		time.Sleep(time.Second * 10)
		log.Fatalln(err.Error())
	}
	defer conn.Close()
	tc := targetinfo.NewTargetServiceClient(conn)

	go c.InfoPushInit(tc)
	go c.HeartBeatInit(tc)

	select {}
}

// InfoPushInit 推送服务情状态
func (c *Client) InfoPushInit(tc targetinfo.TargetServiceClient) {
	allStr, err := tc.TargetInfoReport(context.Background())
	if err != nil {
		log.Println(err.Error())
	}
	go c.InfoPushResultRecv(allStr)
	for {
		cpus, _ := cpu.Counts(true)
		// cpu
		percents, _ := cpu.Percent(time.Second, true)
		var prs []float32
		for _, p := range percents {
			prs = append(prs, float32(p))
		}
		// hostname
		hosts, _ := host.Info()

		// memories
		mems, _ := mem.VirtualMemory()
		// log.Println(mems)

		// disk
		disks, _ := disk.Usage("/")
		diskc, _ := disk.IOCounters()

		// log.Println(diskc)

		ioc, _ := net.IOCounters(false)
		// log.Println(ioc)
		var ns []*targetinfo.NetStatus
		for _, ni := range ioc {
			netInfo := &targetinfo.NetStatus{
				Name:      ni.Name,
				ByteSend:  int64(ni.BytesSent),
				BytesRecv: int64(ni.BytesRecv),
			}
			ns = append(ns, netInfo)
		}

		resp, httpErr := http.Get("http://ip-api.com/json/?lang=zh-CN")
		var ipS struct {
			Status      string  `json:"status"`
			Country     string  `json:"country"`
			CountryCode string  `json:"countryCode"`
			Region      string  `json:"region"`
			RegionName  string  `json:"regionName"`
			City        string  `json:"city"`
			Zip         string  `json:"zip"`
			Lat         float64 `json:"lat"`
			Lon         float64 `json:"lon"`
			Timezone    string  `json:"timezone"`
			Isp         string  `json:"isp"`
			Org         string  `json:"org"`
			As          string  `json:"as"`
			Query       string  `json:"query"`
		}
		if httpErr != nil {
			log.Println("获取ip错误-> ", httpErr.Error())
		} else {
			buffer, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(buffer, &ipS)
			if err != nil {
				log.Println("获取ip错误-> ", err.Error())
			}
		}

		var diskS []*targetinfo.DiskStatus
		for _, v := range diskc {
			diskS = append(diskS, &targetinfo.DiskStatus{
				ReadBytes:  int64(v.ReadBytes),
				WriteBytes: int64(v.WriteBytes),
				ReadCount:  int64(v.ReadCount),
				WriteCount: int64(v.WriteCount),
				Name:       v.Name,
			})
		}

		ti := targetinfo.TargetInfo{
			HostName:       hosts.Hostname,
			OS:             hosts.OS,
			Uptime:         int64(hosts.Uptime),
			BootTime:       int64(hosts.BootTime),
			CPUs:           int32(cpus),
			CPUpr:          prs,
			TotalMem:       int64(mems.Total),
			AvailableMem:   int64(mems.Available),
			UsedPercentMem: float32(mems.UsedPercent),
			UsedMem:        int64(mems.Used),
			FreeMem:        int64(mems.Free),

			TotalDisk:       int64(disks.Total),
			UsedDisk:        int64(disks.Used),
			UsedPercentDisk: int64(disks.UsedPercent),
			FreeDisk:        int64(disks.Free),
			DiskStatus:      diskS,
			NetStatus:       ns,

			Ip:        ipS.Query,
			IpCountry: ipS.Country,
			IpRegion:  ipS.RegionName,

			HostKey: c.ClientConf.ClientKey,

			CheckTime: time.Now().Unix(),
			AuthKey:   c.ClientConf.AuthKey,
		}
		allStr.Send(&ti)
		time.Sleep(time.Second * 4)
	}
}

// InfoPushResultRecv result
func (c *Client) InfoPushResultRecv(allStr targetinfo.TargetService_TargetInfoReportClient) {
	for {
		_, err := allStr.Recv()
		if err != nil {
			time.Sleep(time.Second * 10)
			log.Fatalln(err.Error())
		}
		// log.Println("recv --> ", resp.GetMessage())
	}
}

// HeartBeatInit heart beat
func (c *Client) HeartBeatInit(tc targetinfo.TargetServiceClient) {
	hbClient, err := tc.TargetHeartBeat(context.Background())
	// log.Println("hbClient --> ", hbClient)
	if err != nil {
		log.Println(err.Error())
	}
	go c.HeartBeatResultRecv(hbClient)
	for {
		hbClient.Send(&targetinfo.HeartBeat{
			HostKey:  c.ClientConf.ClientKey,
			BeatTime: time.Now().Unix(),
			AuthKey:  c.ClientConf.AuthKey,
		})
		time.Sleep(time.Second * 1)
	}
}

// HeartBeatResultRecv hb resp
func (c *Client) HeartBeatResultRecv(hc targetinfo.TargetService_TargetHeartBeatClient) {
	for {
		// log.Println("hc -->", hc)
		_, err := hc.Recv()
		if err != nil {
			time.Sleep(time.Second * 10)
			log.Fatalln(err.Error())
		}
		// log.Println(hb.String())
	}
}

var clientConfig = flag.String("f", "./client.yaml", "EyeServer Config Path")

func main() {
	flag.Parse()

	var c Client

	c.InitClient(*clientConfig)

	select {}
}
