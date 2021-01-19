package main

import (
	"flag"
	"log"
	"yogeye/bean"
	"yogeye/targetinfo"
)

var serverConfig = flag.String("f", "./server.yaml", "EyeServer Config Path")

func main() {

	flag.Parse()
	log.Println("config file --> ", *serverConfig)

	serv := bean.EyeServer{
		StatusOfTargets: make(map[string][]targetinfo.TargetInfo),
	}

	serv.InitServ(*serverConfig)

}
