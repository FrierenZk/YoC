package Data

import (
	. "../Debug"
	"../EnvPath"
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
)

var jsonPath = envpath.GetAppDir() + "/json/YoC.json"
var jsonErrEmpty = errors.New("json is empty")

func IsJsonEmpty(err error) bool {
	if err == jsonErrEmpty {
		return true
	}
	return false
}

func checkJsonDir() error {
	var dir, _ = envpath.GetParentDir(jsonPath)
	return envpath.CheckMakeDir(dir)
}

func JsonRead(table *map[string]*stat) error {
	path, err := jsonPath, checkJsonDir()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			defer DebugLogger.Println("json file not exist")
			return jsonErrEmpty
		} else {
			return err
		}
	}
	var scanner = bufio.NewReader(file)
	bytes, err := scanner.ReadBytes('\n')
	for err != io.EOF {
		if err != nil {
			return err
		}
		var dc repository
		err = json.Unmarshal(bytes, &dc)
		if err != nil {
			return err
		}
		(*table)[dc.ID] = new(stat)
		(*table)[dc.ID].Data = &dc
		bytes, err = scanner.ReadBytes('\n')
	}
	defer func() {
		_ = file.Close()
	}()
	return nil
}

func JsonWrite(table *map[string]*stat) error {
	path, err := jsonPath, checkJsonDir()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return err
	}
	for _, ds := range *table {
		if ds == nil {
			continue
		}
		dc, err := json.Marshal(ds.Data)
		if err != nil {
			return err
		}
		dc = append(dc, '\n')
		_, err = file.Write(dc)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadGlobal(global *map[string]string) (err error) {
	var data = make(map[string]string)
	filePath := envpath.GetAppDir()
	filePath += "/YoC.info"
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDONLY, os.ModePerm)
	scanner := bufio.NewReader(file)
	bytes, err := scanner.ReadBytes('\n')
	if err != nil && err != io.EOF {
		log.Println(err)
		return err
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		DebugLogger.Println(err)
	}
	if ps, ok := data["Password"]; !ok || ps == "" {
		data["Password"] = "YoCProject"
	}
	for key, value := range data {
		if _, ok := (*global)[key]; ok {
			if key != "Version" {
				(*global)[key] = value
			}
		}
	}
	if data["Version"] != (*global)["Version"] {
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			log.Println(err)
			return err
		}
		globalInfo, err := json.Marshal(global)
		if err != nil {
			log.Println(err)
			return err
		}
		_, err = file.Write(globalInfo)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
