package main

import (
	"log"
	"os"
	"io"
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	CONFIG_FILE = "web_access_acs_orion.conf"
)

type (
	tomlConfig struct {
		DBPath string
	}
)

func main() {
	setLogging()
	config := readConfig(CONFIG_FILE)

	fmt.Println(config.DBPath)
}

func setLogging() {
	f, err := os.OpenFile(getPath("log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		e(err)
	}
	defer f.Close()
	log.SetOutput(io.MultiWriter(f, os.Stdout))
}

func readConfig(filename string) (config tomlConfig) {
	_, err := toml.DecodeFile(CONFIG_FILE, &config)
	if err != nil {
		e(err)
	}
	return
}

func getPath(newExt string) (path string) {
	base := filepath.Base(os.Args[0])
	ext := filepath.Ext(os.Args[0])
	name := base[:len(base)-len(ext)]
	path = filepath.Dir(os.Args[0]) + string(os.PathSeparator) + name + "." + newExt
	return
}

func e(m interface{}) {
	log.Println(m)
	log.Println("Exit by error")
	os.Exit(1)
}
