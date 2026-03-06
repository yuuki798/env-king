package myutil

import (
	"os"
)

func Mkdir(path string) error {
	//check if the dir exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
