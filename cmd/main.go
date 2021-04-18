package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"randomdcsweather"
	r "randomdcsweather"
	"strings"
	"time"
)

type CloudTemplate struct {
	Preset    *string `json:"preset"`
	Thickness int     `json:"thickness"`
	Density   int     `json:"density"`
	IPRecptns int     `json:"iprecptns"`
	Base      MinMax  `json:"base"`
}

type MinMax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type WeatherSettings struct {
	CloudTemplates []string `json:"cloudTemplates"`
	Times          []struct {
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

func getCloudTemplate(weatherSettings WeatherSettings, baseDir string) (r.CloudTemplate, error) {
	randomSource := rand.NewSource(time.Now().UnixNano())
	random := rand.New(randomSource)
	cloudTemplateIndex := random.Intn(len(weatherSettings.CloudTemplates))
	cloudTemplateByte, err := ioutil.ReadFile(filepath.Join(baseDir, weatherSettings.CloudTemplates[cloudTemplateIndex]))

	if err != nil {
		return r.CloudTemplate{}, err
	}

	cloudTemplate := CloudTemplate{}
	err = json.Unmarshal(cloudTemplateByte, &cloudTemplate)
	if err != nil {
		return r.CloudTemplate{}, err
	}
	base := random.Intn(cloudTemplate.Base.Max-cloudTemplate.Base.Min) + cloudTemplate.Base.Min

	return r.CloudTemplate{
		Preset:    cloudTemplate.Preset,
		Thickness: cloudTemplate.Thickness,
		Density:   cloudTemplate.Density,
		IPRecptns: cloudTemplate.IPRecptns,
		Base:      base,
	}, nil
}

func main() {
	logFile := filepath.Base(os.Args[0]) + ".log"
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Panic(err)
	}
	log.SetOutput(file)

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
		log.Panic(err)
	}

	weatherSettings := WeatherSettings{}
	if err = json.Unmarshal(settingsByte, &weatherSettings); err != nil {
		log.Panic(err)
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
		log.Panic(err)
	}

	cloudTemplate, err := getCloudTemplate(weatherSettings, exePath)
	if err != nil {
		log.Panic(err)
	}

	err = randomdcsweather.SetWeather(missionFile, randomdcsweather.WeatherSettings{
		Day:             missionTime.Day(),
		Month:           int(missionTime.Month()),
		TimeOfDay:       timeOfDay,
		WeatherTemplate: string(templateByte),
		CloudTemplate:   cloudTemplate,
	})

	if err != nil {
		log.Panic(err)
	}
}
