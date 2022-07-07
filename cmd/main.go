package main

import (
	"flag"
	"log"

	"github.com/PanGan21/miniraft/pkg"
)

var (
	serverName = flag.String("server-name", "", "name for the server")
	port       = flag.String("port", "", "port for running the server")
)

func main() {
	flag.Parse()
	validateFlags()

	pkg.StartServer(*&serverName, *&port)
}

func validateFlags() {
	if *serverName == "" {
		log.Fatalf("Must provide serverName for the server")
	}

	if *port == "" {
		log.Fatalf("Must provide a port number for server to run")
	}
}
