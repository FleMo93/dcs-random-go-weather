package dynamicgoweather

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

func zipDirectory(source string, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		return errors.New("Source is no directory")
	}

	if source[len(source)-1] != '\\' {
		source += "\\"
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if source == path {
			return nil
		}
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, source)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func getWeather(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	templateDir := filepath.Join(dir, "templates")
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
	fmt.Println(mission)
	err = os.WriteFile(missionFilePath, []byte(mission), os.ModeDevice)
	return err
}

// SetWeather sets the weather of a DCS mission file
func SetWeather(mizFile string, weatherName string) error {
	weatherTemplate, err := getWeather(weatherName)

	if err != nil {
		return err
	}

	dir, err := os.Getwd()
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

	_, filename := filepath.Split(mizFile)
	trgt := filepath.Join(dir, filename)
	zipDirectory(extractDir, trgt)
	zipDirectory(extractDir, filename+".zip")
	err = os.RemoveAll(extractDir)
	if err != nil {
		return err
	}

	return nil
}
