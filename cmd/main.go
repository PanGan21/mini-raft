package main

import (
	"errors"
	"flag"
	"log"
)

var VALIDATION_ERROR = errors.New("validation error; flags cannot have the default values")

func main() {
	port := flag.Int("port", 0, "rpc listen port")
	peer_addresses := flag.String("peer_addresses", "", "comma separated peer addresses")
	id := flag.Int("id", 0, "node id")

	flag.Parse()

	err := validate_flags(*port, *peer_addresses, *id)
	if err != nil {
		log.Fatal(err)
	}
}

func validate_flags(port int, peer_addresses string, id int) error {
	if port == 0 || id == 0 || peer_addresses == "" {
		return VALIDATION_ERROR
	}

	return nil
}
