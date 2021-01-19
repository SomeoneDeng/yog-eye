package main

import (
	"flag"
	"log"
	"yogeye/bean"
)

var serverConfig = flag.String("f", "./server.yaml", "EyeServer Config Path")

func main() {

	flag.Parse()
	log.Println("config file --> ", *serverConfig)

	serv := bean.EyeServer{}

	serv.InitServ(*serverConfig)

}
