package storage

import (
	"bufio"
	"os"
)

func createFileIfNotExists(fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func writeToFile(fileName string, msg string) error {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = file.WriteString(msg)
	if err != nil {
		return err
	}
	return nil
}

func readFile(fileName string) ([]string, error) {
	var rows []string

	file, err := os.Open(fileName)
	if err != nil {
		return rows, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rows = append(rows, scanner.Text())
	}
	return rows, nil
}
