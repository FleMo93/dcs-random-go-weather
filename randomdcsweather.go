package randomdcsweather

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mholt/archiver"
)

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

func getWeather(name string) (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	templateDir := filepath.Join(dir, "weather-templates")
	dirEntries, err := os.ReadDir(templateDir)
	if err != nil {
		return "", err
	}

	weatherFile := ""

	if name != "" {
		weatherFile = filepath.Join(templateDir, name)
	} else {
		rand.Seed(time.Now().UnixNano())
		index := rand.Intn(len(dirEntries))
		file := dirEntries[index]
		weatherFile = filepath.Join(templateDir, file.Name())
	}

	_, err = os.Stat(weatherFile)
	if err != nil || weatherFile == "" {
		return "", errors.New("Template weather file not found")
	}

	fileByte, err := os.ReadFile(weatherFile)

	if err != nil {
		return "", err
	}

	return string(fileByte), nil
}

func setWeather(missionFilePath string, weather string) error {
	fileByte, err := os.ReadFile(missionFilePath)
	if err != nil {
		return err
	}

	mission := string(fileByte)
	re := regexp.MustCompile(`(?s)\["weather"\].*(?:end of \["weather"\])`)
	mission = re.ReplaceAllString(mission, weather)
	if re.FindString(mission) == "" {
		return errors.New("Not found")
	}

	err = os.WriteFile(missionFilePath, []byte(mission), os.ModeDevice)
	return err
}

// SetWeather sets the weather of a DCS mission file
func SetWeather(mizFile string, weatherName string) error {
	weatherTemplate, err := getWeather(weatherName)

	if err != nil {
		return err
	}

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

	err = setWeather(missionFile, weatherTemplate)

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
