package main

import (
	"dynamicgoweather"
	"log"
	"os"
	"strings"
)

func main() {
	arg := os.Args
	var missionFile = ""
	var weather = ""
	for _, ele := range arg {
		if strings.Index(ele, "-m ") == 0 {
			missionFile = ele[3:]
		} else if strings.Index(ele, "-w ") == 0 {
			weather = ele[3:]
		}
	}

	if missionFile == "" {
		log.Fatal("Missing \"--m\" parameter")
		os.Exit(1)
	}

	err := dynamicgoweather.SetWeather(missionFile, weather)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
