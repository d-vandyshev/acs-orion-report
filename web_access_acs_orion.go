package main

import (
	"log"
	"os"
	"io"
	"fmt"
	"path/filepath"
	"github.com/BurntSushi/toml"
	"net/http"
	auth "github.com/abbot/go-http-auth"
)

const (
	CONFIG_FILE = "web_access_acs_orion.conf"
)

type (
	// Configuration
	tomlConfig struct {
		OrionDatabase OrionDatabaseInfo
		WebServer webServerInfo
	}
	OrionDatabaseInfo struct {
		Path string
	}
	webServerInfo struct {
		AuthUsername string
		AuthPassword string
	}
)

var config tomlConfig

func main() {
	setLogging()
	config = readConfig(CONFIG_FILE)

	authenticator := auth.NewBasicAuthenticator(
		"example.com",
		func(user, realm string) string {
			return Secret(config, user, realm)
		})

	http.HandleFunc("/", authenticator.Wrap(handle))
	http.ListenAndServe(":8080", nil)

}

func handle(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	fmt.Fprintf(w, "<html><body><h1>Hello, %s!</h1></body></html>", r.Username)
}

func Secret(config tomlConfig, user, realm string) string {
	if user == config.WebServer.AuthUsername {
		return string(auth.MD5Crypt([]byte(config.WebServer.AuthPassword), []byte("J.w7a-.1"), []byte("$apr1$")))
	}
	return ""
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
		fmt.Println(err)
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
