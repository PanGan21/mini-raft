package storage

import (
	"errors"
	"strconv"
	"strings"
)

type Database struct {
	db map[string]int
}

func NewDatabase() (db *Database) {
	kv_store := make(map[string]int)
	return &Database{db: kv_store}
}

func (d *Database) setByKey(key string, value int) {
	d.db[key] = value
}

func (d *Database) getByKey(key string) (int, error) {
	value, exists := d.db[key]
	if !exists {
		return -1, errors.New("key not found")
	}
	return value, nil
}

func (d *Database) deleteByKey(key string) error {
	_, exists := d.db[key]
	if !exists {
		return errors.New("key not found")
	}
	delete(d.db, key)
	return nil
}

func (d *Database) ValidateCommand(command string) error {
	splittedCommand := strings.Split(command, " ")
	operation := splittedCommand[0]
	if operation == "GET" || operation == "DELETE" {
		if len(splittedCommand) != 2 {
			return errors.New("need a key for GET/DELETE operation")
		}
	} else if operation == "SET" {
		if len(splittedCommand) != 3 {
			return errors.New("need a key and a value for SET operation")
		}
		_, err := strconv.Atoi(splittedCommand[2])
		if err != nil {
			return errors.New("not a valid integer value")
		}
	} else {
		return errors.New("invalid command")
	}
	return nil
}

func (d *Database) PerformDbOperations(command string) string {
	splittedCommand := strings.Split(command, " ")
	operation := splittedCommand[0]
	var response string = ""
	if operation == "GET" {
		key := splittedCommand[1]
		val, err := d.getByKey(key)
		if err != nil {
			response = "Key not found error"
		} else {
			response = "Value for key (" + key + ") is: " + strconv.Itoa(val)
		}
	} else if operation == "SET" {
		key := splittedCommand[1]
		val, _ := strconv.Atoi(splittedCommand[2])
		if response == "" {
			d.setByKey(key, val)
		}
		if response == "" {
			response = "Key set successfully"
		}
	} else if operation == "DELETE" {
		key := splittedCommand[1]
		if err := d.deleteByKey(key); err != nil {
			response = "Key not found"
		}
		if response == "" {
			response = "Key deleted successfully"
		}
	}
	return response
}

func (d *Database) LogCommand(command string, serverName string) error {
	fileName := serverName + ".txt"
	var err = createFileIfNotExists(fileName)
	if err != nil {
		return err
	}
	err = writeToFile(fileName, serverName+","+command+"\n")
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) RebuildLogIfExists(serverName string) []string {
	logs := make([]string, 0)
	fileName := serverName + ".txt"
	createFileIfNotExists(fileName)
	lines, _ := readFile(fileName)
	for _, line := range lines {
		splittedCommand := strings.Split(line, ",")
		logs = append(logs, splittedCommand[1])
	}
	return logs
}
