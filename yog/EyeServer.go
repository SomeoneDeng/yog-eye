package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"yogeye/targetinfo"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type JwtConfig struct {
	SignKey string `yaml:"sign_key"`
	// day
	Expire int `yaml:"expire"`

	AdminKey string `yaml:"admin_key"`
}

type ServerConfig struct {
	Port         string `yaml:"port"`
	HTTPPort     string `yaml:"http_port"`
	ClientExpire int    `yaml:"client_expire"`
	// client password
	Auth string `yaml:"auth"`

	JwtCfg JwtConfig `yaml:"jwt"`
}

type EyeServer struct {
	Conf ServerConfig

	// StatusOfTargets   target1_name -> [status],
	StatusOfTargets map[string][]targetinfo.TargetInfo

	HeartBeatMap map[string]int64

	// for rest data interface
	HTTPServ *gin.Engine
}

type StatusResp struct {
	Name string                  `json:"name"`
	List []targetinfo.TargetInfo `json:"list"`
}

type HeartBeatResp struct {
	Name     string `json:"name"`
	BeatTime int64  `json:"beat_time"`
}

// TargetInfoReport accept machine status from client
func (es *EyeServer) TargetInfoReport(ts targetinfo.TargetService_TargetInfoReportServer) error {
	for {
		ti, err := ts.Recv()
		if err != nil {
			time.Sleep(time.Second * 10)
			log.Println(err.Error())
			continue
		}
		// log.Println("from --> ", ti.HostKey, " --> ", ti.GetCPUpr())

		if ti.AuthKey != es.Conf.Auth {
			ts.Send(&targetinfo.Response{
				Message: "error auth",
			})
			continue
		}

		es.StatusOfTargets[ti.HostKey] = append(es.StatusOfTargets[ti.HostKey], *ti)
		if len(es.StatusOfTargets[ti.HostKey]) == 10 {
			es.StatusOfTargets[ti.HostKey] = es.StatusOfTargets[ti.HostKey][1:]
		}

		ts.Send(&targetinfo.Response{
			Message: strconv.Itoa((int(ti.GetCPUs()) + 1)),
		})
	}
}

// TargetHeartBeat heart beat recv
func (es *EyeServer) TargetHeartBeat(hb targetinfo.TargetService_TargetHeartBeatServer) error {
	for {
		hearBeat, err := hb.Recv()
		if err != nil {
			time.Sleep(time.Second * 10)
			log.Println(err.Error())
			continue
		}
		if hearBeat.AuthKey != es.Conf.Auth {
			hearBeat.BeatTime = time.Now().Unix()
			hb.Send(hearBeat)
			continue
		}
		// log.Println("收到心跳 --> ", hearBeat.GetHostKey(), "[", hearBeat.GetBeatTime(), "]")
		es.HeartBeatMap[hearBeat.GetHostKey()] = hearBeat.GetBeatTime()
		hearBeat.BeatTime = time.Now().Unix()
		hb.Send(hearBeat)
	}
}

func (es *EyeServer) readConfig(config string) {
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
	es.Conf = cnf
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
				// delete(es.StatusOfTargets, k)
			}
		}
		// check every 1 sec
		time.Sleep(time.Second * 1)
	}
}

// InitServ init whole server
func (es *EyeServer) InitServ(configPath string) {

	go es.InitHeartBeatManager()

	es.readConfig(configPath)
	go es.InitHttpServ()

	// open rpc port
	lis, err := net.Listen("tcp", es.Conf.Port)
	if err != nil {
		log.Println(err.Error())
		return
	}
	s := grpc.NewServer()
	targetinfo.RegisterTargetServiceServer(s, es)
	s.Serve(lis)
}

// InitHttpServ rest server
func (es *EyeServer) InitHttpServ() {
	es.HTTPServ = gin.Default()
	es.HTTPServ.Use(cors.Default())
	log.Println("http port --> ", es.Conf.HTTPPort)

	es.HTTPServ.Static("/yog/static", "./frontend")
	es.HTTPServ.Use(func(c *gin.Context) {
		log.Println(c.Request.URL)

		if strings.HasSuffix(c.Request.URL.Path, "login") || strings.HasSuffix(c.Request.URL.Path, "login.html") {
			c.Next()
			return
		}
		authJwt, err := c.Cookie("auth")
		if err != nil {
			log.Println("to login")
			log.Println(err.Error())
			c.JSON(http.StatusOK, 403)
			c.Abort()
			return
		}
		if authJwt == "" {
			log.Println("to login")
			c.Redirect(http.StatusTemporaryRedirect, "/yog/static/login.html")
			return
		}

		tk, err := jwt.Parse(authJwt, func(t *jwt.Token) (interface{}, error) {
			return []byte(es.Conf.JwtCfg.SignKey), nil
		})
		if err != nil {
			log.Println("to login")
			log.Println(err.Error())
			c.JSON(http.StatusOK, 403)
			c.Abort()
			return
		}
		log.Println(tk.Claims)

		c.Next()
	})

	yog := es.HTTPServ.Group("/yog")
	{
		yog.GET("/status", func(c *gin.Context) {

			var statusResp []StatusResp
			for k, v := range es.StatusOfTargets {
				var stastatus StatusResp
				stastatus.Name = k
				stastatus.List = v
				statusResp = append(statusResp, stastatus)
			}

			c.JSON(http.StatusOK, statusResp)
		})

		yog.GET("/hearts", func(c *gin.Context) {
			var hbResp []HeartBeatResp
			for k, v := range es.HeartBeatMap {
				var hb HeartBeatResp
				hb.Name = k
				hb.BeatTime = v
				hbResp = append(hbResp, hb)
			}
			c.JSON(http.StatusOK, hbResp)
		})

		yog.GET("/login", func(c *gin.Context) {
			c.Request.ParseForm()
			adminKey, _ := c.GetQuery("adminKey")
			log.Println("adminKey --> ", adminKey, time.Now().Add(time.Hour*24*time.Duration(es.Conf.JwtCfg.Expire)).Unix())

			if adminKey == es.Conf.JwtCfg.AdminKey {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
					Issuer:    "admin",
					ExpiresAt: time.Now().Add(time.Hour * 24 * time.Duration(es.Conf.JwtCfg.Expire)).Unix(),
					Subject:   "user",
				})
				tokenString, err := token.SignedString([]byte(es.Conf.JwtCfg.SignKey))
				if err != nil {
					log.Println(err.Error())
					c.JSON(http.StatusInternalServerError, make(map[string]string, 0))
				}
				c.SetCookie("auth", tokenString, 7*24*60*60, "/", "", false, true)
				c.Redirect(http.StatusPermanentRedirect, "static/index.html")
				// c.SetCookie("auth", tokenString, int(time.Hour)*24*7, "/", "", false, true)
			} else {
				c.Redirect(http.StatusPermanentRedirect, "static/login.html")
			}
		})
	}

	err := es.HTTPServ.Run(es.Conf.HTTPPort)
	if err != nil {
		log.Println(err.Error())
	}
}

var serverConfig = flag.String("f", "./server.yaml", "EyeServer Config Path")

func main() {

	flag.Parse()
	log.Println("config file --> ", *serverConfig)

	serv := EyeServer{
		StatusOfTargets: make(map[string][]targetinfo.TargetInfo),
	}

	serv.InitServ(*serverConfig)

}
