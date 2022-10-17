package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
)

type ServerConf struct {
	Host string `yaml:"server-host"`
	Port string `yaml:"server-host"`
}

func main() {
	yamlFile, err := ioutil.ReadFile("server_conf.yml")
	if err != nil {
		fmt.Println("Could not read server_conf.yml")
		return
	}

	var serverConf ServerConf
	err = yaml.Unmarshal(yamlFile, &serverConf)
	if err != nil {
		fmt.Println("Could not parse server_conf.yml")
		return
	}

	fs := http.FileServer(http.Dir("../static"))
	http.Handle("/", fs)

	log.Print("Listening on host " + serverConf.Host + " on port " + serverConf.Port)
	err = http.ListenAndServe(serverConf.Host+":"+serverConf.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
