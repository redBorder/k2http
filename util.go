package main

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
)

// Config store the main configuration for the application
type Config struct {
	Backend map[string]interface{} `yaml:"backend"`
	Kafka   map[string]interface{} `yaml:"kafka"`
	HTTP    map[string]interface{} `yaml:"http"`
}

// LoadConfigFile open an YAML file and parse the content
func LoadConfigFile(fileName string) (config Config, err error) {
	configData, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	if err = yaml.Unmarshal([]byte(configData), &config); err != nil {
		return
	}

	return
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("\n%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("\n%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

func recoverPanic() {
	r := recover()
	if r == nil {
		return
	}
	logrus.Fatal("Error: ", r, identifyPanic())
}
