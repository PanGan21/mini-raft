package storage

import (
	"errors"
	"strconv"
	"strings"
)

const registryFileName string = "registry.txt"
const stateFileName string = "state.txt"

func RegisterServer(serverName string, port string) error {
	err := createFileIfNotExists(registryFileName)
	if err != nil {
		return err
	}
	log := serverName + "," + port + "\n"
	err = writeToFile(registryFileName, log)
	if err != nil {
		return err
	}
	return nil
}

func GetRegisteredServers() (map[string]int, error) {
	s := make(map[string]int)
	rows, err := readFile(registryFileName)
	if err != nil {
		return s, err
	}
	for _, row := range rows {
		splitted := strings.Split(row, ",")
		port, _ := strconv.Atoi(splitted[1])
		s[splitted[0]] = port
	}
	return s, nil
}

func PersistState(log string) error {
	err := createFileIfNotExists(stateFileName)
	if err != nil {
		return err
	}
	err = writeToFile(stateFileName, log+"\n")
	if err != nil {
		return err
	}
	return nil
}

func GetCurrentStateByServer(serverName string) (string, error) {
	logs, err := readFile(stateFileName)
	if err != nil {
		return "", err
	}

	var currentLog = ""
	for _, row := range logs {
		splits := strings.Split(row, ",")
		if splits[0] == serverName {
			currentLog = row
		}
	}
	if currentLog == "" {
		return "", errors.New("state not found for specified server")
	}
	return currentLog, nil
}
