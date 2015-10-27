package main

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	auth "github.com/abbot/go-http-auth"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type (
	// Configuration
	tomlConfig struct {
		OrionDatabase OrionDatabaseInfo
		WebServer     webServerInfo
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
	config = readConfig(getPath("conf"))

	authenticator := auth.NewBasicAuthenticator(
		"example.com",
		func(user, realm string) string {
			return Secret(config, user, realm)
		})

	http.HandleFunc("/", authenticator.Wrap(handle))
	http.HandleFunc("/last-seen-employees", authenticator.Wrap(showLastSeenEmployees))
	http.ListenAndServe(":8080", nil)

}

type (
	lastSeenEmployee struct {
		TabNum int
		Name   string
		Time   time.Time
	}

	lastSeenEmployeesList []lastSeenEmployee
)

func showLastSeenEmployees(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	emps := new(employeesTable)
	err := paradoxReadTable(config.OrionDatabase.Path+string(os.PathSeparator)+"pList.DB", emps, 0)
	if err != nil {
		http.Error(w, "Error 101 accessing database Paradox: "+err.Error(), 500)
		return
	}
	sort.Sort(emps)
	// All events
	events := new(eventsTable)
	err = paradoxReadTable(config.OrionDatabase.Path+string(os.PathSeparator)+"pLogData.db", events, 0)
	if err != nil {
		http.Error(w, "Error 102 accessing database Paradox: "+err.Error(), 500)
		return
	}
	sort.Sort(events)
	var items lastSeenEmployeesList
	for _, emp := range *emps {
		for _, ev := range *events {
			if emp.ID == ev.HozOrgan {
				//				fmt.Println(emp.Name, emp.FirstName, emp.MidName, ";", ev.TimeVal)
				items = append(items, lastSeenEmployee{emp.TabNumber, emp.Name + emp.FirstName + emp.MidName, ev.TimeVal})
				break
			}
		}
	}
	jsonData, err := json.Marshal(items)
	if err != nil {
		http.Error(w, "Error 103 converting data in json: "+err.Error(), 500)
		return
	}
	fmt.Fprintf(w, string(jsonData))
}

//
// Employees
//
type employee struct {
	ID        int
	TabNumber int
	Name      string
	FirstName string
	MidName   string
	Status    int
	WorkPhone string
	HomePhone string
	Picture   string
	Birthdate string
	Address   string
	Section   int
	Post      string
	Schedule  int
	Company   string
	Avto      string
	Spack     int
	Weight    int
	Deviation int
	Config    int
}

type employeesTable []employee

func (et *employeesTable) appendRow(values ...interface{}) {
	var e employee
	e.ID = values[0].(int)
	e.TabNumber = values[1].(int)
	e.Name = values[2].(string)
	e.FirstName = values[3].(string)
	e.MidName = values[4].(string)
	e.Status = values[5].(int)
	e.WorkPhone = values[6].(string)
	e.HomePhone = values[7].(string)
	e.Picture = values[8].(string)
	e.Birthdate = values[9].(string)
	e.Address = values[10].(string)
	e.Section = values[11].(int)
	e.Post = values[12].(string)
	e.Schedule = values[13].(int)
	e.Company = values[14].(string)
	e.Avto = values[15].(string)
	e.Spack = values[16].(int)
	e.Weight = values[17].(int)
	e.Deviation = values[18].(int)
	e.Config = values[19].(int)
	*et = append(*et, e)
}

// For sort by Name
func (slice employeesTable) Len() int {
	return len(slice)
}
func (slice employeesTable) Less(i, j int) bool {
	return slice[i].Name < slice[j].Name
}
func (slice employeesTable) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

//
// Events
//
type event struct {
	Num             int
	TimeVal         time.Time
	DoorIndex       int
	HozOrgan        int
	RazdIndex       int
	ReaderIndex     int
	Shleif          int
	AccessZoneIndex int
	Event           int
	IndexKey        int
	Remark          string
	NetPrz          int
	ADC             int
	Mode            int
	DeviceTime      time.Time
	Sign            int
}

type eventsTable []event

func (et *eventsTable) appendRow(values ...interface{}) {
	var e event
	e.Num = values[0].(int)
	e.TimeVal = values[1].(time.Time)
	e.DoorIndex = values[2].(int)
	e.HozOrgan = values[3].(int)
	e.RazdIndex = values[4].(int)
	e.ReaderIndex = values[5].(int)
	e.Shleif = values[6].(int)
	e.AccessZoneIndex = values[7].(int)
	e.Event = values[8].(int)
	e.IndexKey = values[9].(int)
	e.Remark = values[10].(string)
	e.NetPrz = values[11].(int)
	e.ADC = values[12].(int)
	e.Mode = values[13].(int)
	e.DeviceTime = values[14].(time.Time)
	e.Sign = values[15].(int)
	if e.Event == 28 {
		*et = append(*et, e)
	}
}

// For sort by TimeVal
func (slice eventsTable) Len() int {
	return len(slice)
}
func (slice eventsTable) Less(i, j int) bool {
	return slice[j].TimeVal.Before(slice[i].TimeVal)
	//	return slice[i].TimeVal.After(slice[j].TimeVal)
}
func (slice eventsTable) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
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
	_, err := toml.DecodeFile(filename, &config)
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
