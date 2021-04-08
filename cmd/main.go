package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"randomdcsweather"
	"strings"
	"time"
)

type WeatherSettings struct {
	Times []struct {
		Date struct {
			Start struct {
				Day   int `json:"day"`
				Month int `json:"month"`
			} `json:"start"`
			End struct {
				Day   int `json:"day"`
				Month int `json:"month"`
			} `json:"end"`
		} `json:"date"`
		TimeOfDay struct {
			Earliest int `json:"earliest"`
			Latest   int `json:"latest"`
		} `json:"timeOfDay"`
		WeatherTemplates []string `json:"weatherTemplates"`
	} `json:"times"`
}

func main() {
	arg := os.Args
	var missionFile = ""
	for _, ele := range arg {
		if strings.Index(ele, "-m ") == 0 {
			missionFile = ele[3:]
		}
	}

	exePath := filepath.Dir(arg[0])
	settingsByte, err := os.ReadFile(filepath.Join(exePath, "./settings.json"))

	if err != nil {
		log.Fatal(err)
	}

	weatherSettings := WeatherSettings{}
	if err = json.Unmarshal(settingsByte, &weatherSettings); err != nil {
		log.Fatal(err)
	}

	randomSource := rand.NewSource(time.Now().UnixNano())
	random := rand.New(randomSource)
	weatherIndex := random.Intn(len(weatherSettings.Times))
	weather := weatherSettings.Times[weatherIndex]
	timeOfDay := random.Intn((weather.TimeOfDay.Latest - weather.TimeOfDay.Earliest)) + weather.TimeOfDay.Earliest

	minDate := time.Date(2000, time.Month(weather.Date.Start.Month), weather.Date.Start.Day, 0, 0, 0, 0, time.UTC).Unix()
	maxDate := time.Date(2000, time.Month(weather.Date.End.Month), weather.Date.End.Day, 0, 0, 0, 0, time.UTC).Unix()
	deltaDate := rand.Int63n(maxDate-minDate) + minDate
	missionTime := time.Unix(deltaDate, 0)

	weatherTemplateIndex := random.Intn(len(weather.WeatherTemplates))
	templateByte, err := ioutil.ReadFile(filepath.Join(exePath, weather.WeatherTemplates[weatherTemplateIndex]))
	if err != nil {
		log.Fatal(err)
	}

	err = randomdcsweather.SetWeather(missionFile, randomdcsweather.WeatherSettings{
		Day:             missionTime.Day(),
		Month:           int(missionTime.Month()),
		TimeOfDay:       timeOfDay,
		WeatherTemplate: string(templateByte),
	})

	if err != nil {
		log.Fatal(err)
	}
}
