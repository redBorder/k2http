package main

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder"
)

// Config store the main configuration for the application
type Config struct {
	Backend map[string]interface{} `yaml:"backend"`
	Kafka   map[string]interface{} `yaml:"kafka"`
	HTTP    map[string]interface{} `yaml:"http"`
}

// LoadConfigFile open an YAML file and parse the content
func LoadConfigFile(fileName string) (rbForwarderConfig rbforwarder.Config,
	KafkaConfig, HTTPConfig map[string]interface{}, err error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}
	config := Config{}
	if err = yaml.Unmarshal([]byte(data), &config); err != nil {
		return
	}

	KafkaConfig = config.Kafka
	HTTPConfig = config.HTTP

	// Get number of workers
	if workers, ok := config.Backend["workers"].(int); ok {
		rbForwarderConfig.Workers = workers
	} else {
		rbForwarderConfig.Workers = defaultWorkers
	}

	// Get number of retries per message
	if retries, ok := config.Backend["retries"].(int); ok {
		rbForwarderConfig.Retries = retries
	} else {
		rbForwarderConfig.Retries = defaultRetries
	}

	// Time to wait between retries
	if backoff, ok := config.Backend["backoff"].(int); ok {
		rbForwarderConfig.Backoff = backoff
	} else {
		rbForwarderConfig.Backoff = defaultBackoff
	}

	// Get queue size
	if queue, ok := config.Backend["queue"].(int); ok {
		rbForwarderConfig.QueueSize = queue
	} else {
		rbForwarderConfig.QueueSize = defaultQueueSize
	}

	// Get max message rate
	if maxMessages, ok := config.Backend["max_messages"].(int); ok {
		rbForwarderConfig.MaxMessages = maxMessages
	}

	// Get max bytes rate
	if maxBytes, ok := config.Backend["max_bytes"].(int); ok {
		rbForwarderConfig.MaxBytes = maxBytes
	}

	// Get the interval to show message rate
	if interval, ok := config.Backend["showcounter"].(int); ok {
		rbForwarderConfig.ShowCounter = interval
	}

	// Show debug info
	if *debug {
		rbForwarderConfig.Debug = true
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
