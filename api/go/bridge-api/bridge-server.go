package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/amimof/huego"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"image/color"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"
)

var serverConf ServerConf

type ServerConf struct {
	Host       string `yaml:"server-host"`
	Port       string `yaml:"server-port"`
	BridgeHost string `yaml:"bridge-host"`
	BridgeId   string `yaml:"bridge-id"`
}

func handleBrightness(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	level := mux.Vars(r)["level"]

	bridge := huego.New(serverConf.BridgeHost, serverConf.BridgeId)
	lights, err := bridge.GetLights()
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not connect to bridge to get the lights on the network."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	index, _ := strconv.Atoi(id)
	if index > len(lights) {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Index provided is greater than the amount of lights on the network."))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	brightness, _ := strconv.Atoi(level)
	if brightness < 0 || brightness > 254 {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Brightness level is not between [0-254]."))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	light, err := bridge.GetLight(index)
	err = light.Bri(uint8(brightness))
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not change the brightness on light."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleChangeColor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	rgbStr := mux.Vars(r)["rgb"]

	re := regexp.MustCompile("-")
	rgbSplit := re.Split(rgbStr, -1)
	var rgb []int

	for _, value := range rgbSplit {
		value, _ := strconv.Atoi(value)
		if value < 0 || value > 255 {
			w.Header().Set("Content-Type", "application/text")
			w.Write([]byte("An RGB value provided is not between [0-255]."))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rgb = append(rgb, value)
	}

	bridge := huego.New(serverConf.BridgeHost, serverConf.BridgeId)
	lights, err := bridge.GetLights()
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not connect to bridge to get the lights on the network."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	index, _ := strconv.Atoi(id)
	if index > len(lights) {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Index provided is greater than the amount of lights on the network."))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	light, err := bridge.GetLight(index)

	err = light.Col(&color.RGBA{
		R: uint8(rgb[0]),
		G: uint8(rgb[1]),
		B: uint8(rgb[2]),
		A: uint8(255),
	})

	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not change the color on light."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleGetLights(w http.ResponseWriter, r *http.Request) {
	bridge := huego.New(serverConf.BridgeHost, serverConf.BridgeId)
	lights, err := bridge.GetLights()
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not connect to bridge to get the lights on the network."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	lightsStream, err := json.Marshal(lights)

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(lightsStream)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleGetLight(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	bridge := huego.New(serverConf.BridgeHost, serverConf.BridgeId)
	lights, err := bridge.GetLights()
	if err != nil {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Could not connect to bridge to get the lights on the network."))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	index, _ := strconv.Atoi(id)
	if index > len(lights) {
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Index provided is greater than the amount of lights on the network."))
		w.WriteHeader(http.StatusBadRequest)
		return
		return
	}
	lightsStream, err := json.Marshal(lights[index])

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(lightsStream)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	yamlFile, err := ioutil.ReadFile("server-conf.yml")
	if err != nil {
		log.Fatalln("Could not read server-conf.yml file.")
	}

	err = yaml.Unmarshal(yamlFile, &serverConf)
	if err != nil {
		log.Fatalln("Could not parse server-conf.yml file.")
	}

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	/* Http routes */
	r := mux.NewRouter()
	r.HandleFunc("/api/lights/", handleGetLights)
	r.HandleFunc("/api/lights/{id:[0-9]+}", handleGetLight)
	r.HandleFunc("/api/lights/{id:[0-9]+}/brightness/{level:[0-9]+}", handleBrightness)
	r.HandleFunc("/api/lights/{id:[0-9]+}/color/{rgb:[-0-9]+}", handleChangeColor)

	srv := &http.Server{
		Handler:      r,
		Addr:         serverConf.Host + ":" + serverConf.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Serving on " + serverConf.Host + " on port " + serverConf.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
