package randomdcsweather

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mholt/archiver"
	lua "github.com/yuin/gopher-lua"
)

type WeatherSettings struct {
	Day             int
	Month           int
	TimeOfDay       int
	WeatherTemplate string
	CloudTemplate   CloudTemplate
}

type CloudTemplate struct {
	Preset    *string `json:"preset"`
	Thickness int     `json:"thickness"`
	Density   int     `json:"density"`
	IPRecptns int     `json:"iprecptns"`
	Base      int     `json:"base"`
}

type mission struct {
	date struct {
		year  int
		day   int
		month int
	}
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func archiverDirectory(source string, target string) error {
	filesList := []string{}

	fileInfos, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	for _, fi := range fileInfos {
		filesList = append(filesList, source+"\\"+fi.Name())
	}

	zip := archiver.NewZip()
	err = zip.Archive(filesList, target)
	if err != nil {
		return err
	}

	return nil
}

func setWeather(missionFilePath string, weatherTemplate string) error {
	fileByte, err := os.ReadFile(missionFilePath)
	if err != nil {
		return err
	}

	mission := string(fileByte)
	re := regexp.MustCompile(`(?s)\["weather"\].*(?:end of \["weather"\])`)
	mission = re.ReplaceAllString(mission, weatherTemplate)
	if re.FindString(mission) == "" {
		return errors.New("Not found")
	}

	err = os.WriteFile(missionFilePath, []byte(mission), os.ModeDevice)
	return err
}

func setTime(lMission *lua.LTable, time int) error {
	if lMission.RawGetString("start_time") == nil {
		return errors.New("Mission table has no \"start_time\" key")
	}
	lMission.RawSetString("start_time", lua.LNumber(time))
	return nil
}

func setDate(lMission *lua.LTable, month int, day int) error {
	dateTable := lMission.RawGetString("date").(*lua.LTable)
	if dateTable == nil {
		return errors.New("Date table not found")
	}

	if dateTable.RawGetString("Month") == nil {
		return errors.New("Date table has no \"Month\" key")
	}

	if dateTable.RawGetString("Day") == nil {
		return errors.New("Date table has no \"Day\" key")
	}

	dateTable.RawSetString("Month", lua.LNumber(month))
	dateTable.RawSetString("Day", lua.LNumber(day))

	return nil
}

func setClouds(lMission *lua.LTable, cloudTemplate CloudTemplate) error {
	weatherTable := lMission.RawGetString("weather").(*lua.LTable)
	if weatherTable == nil {
		return errors.New("Weather table not found")
	}

	cloudsTable := weatherTable.RawGetString("clouds").(*lua.LTable)
	if cloudsTable == nil {
		return errors.New("Clouds table not found")
	}

	if cloudTemplate.Preset != nil {
		cloudsTable.RawSetString("preset", lua.LString(*cloudTemplate.Preset))
	} else {
		cloudsTable.RawSetString("preset", lua.LNil)
	}

	cloudsTable.RawSetString("thickness", lua.LNumber(cloudTemplate.Thickness))
	cloudsTable.RawSetString("density", lua.LNumber(cloudTemplate.Density))
	cloudsTable.RawSetString("iprecptns", lua.LNumber(cloudTemplate.IPRecptns))
	cloudsTable.RawSetString("base", lua.LNumber(cloudTemplate.Base))

	return nil
}

// SetWeather sets the weather of a DCS mission file
func SetWeather(mizFile string, weather WeatherSettings) error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}

	extractDir := filepath.Join(dir, "extract")
	res, err := unzip(mizFile, extractDir)
	if err != nil {
		return err
	}

	missionFile := ""
	for _, f := range res {
		_, filename := filepath.Split(f)
		if filename == "mission" {
			missionFile = f
		}
	}

	if missionFile == "" {
		return errors.New("Mission file in .miz not found")
	}

	if err = setWeather(missionFile, weather.WeatherTemplate); err != nil {
		return err
	}

	missionBytes, err := ioutil.ReadFile(missionFile)
	if err != nil {
		return err
	}

	missionString := string(missionBytes)
	l := lua.NewState()
	defer l.Close()
	err = l.DoString(missionString)
	if err != nil {
		return err
	}

	lMission := l.GetGlobal("mission").(*lua.LTable)

	if err = setDate(lMission, weather.Month, weather.Day); err != nil {
		return err
	}

	if err = setTime(lMission, weather.TimeOfDay); err != nil {
		return err
	}

	if err = setClouds(lMission, weather.CloudTemplate); err != nil {
		return err
	}

	str := luaTableToString("mission", lMission)
	err = ioutil.WriteFile(missionFile, []byte(str+"\n"), 0644)
	if err != nil {
		return err
	}

	zipPath := mizFile + ".tmp.zip"
	os.Remove(zipPath)
	err = archiverDirectory(extractDir, zipPath)
	os.Rename(zipPath, mizFile)
	if err != nil {
		return err
	}

	err = os.RemoveAll(extractDir)
	if err != nil {
		return err
	}

	return nil
}
