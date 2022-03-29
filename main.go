// Copyright (C) 2016 Eneo Tecnologia S.L.
// Diego Fernández Barrera <bigomby@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	"github.com/bsm/sarama-cluster"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/components/batch"
	"github.com/redBorder/rbforwarder/components/httpsender"
	"github.com/redBorder/rbforwarder/components/limiter"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	defaultQueueSize = 10000
	defaultWorkers   = 1
	defaultRetries   = 0
	defaultBackoff   = 2
)

// Logger is the main logger object
var Logger = logrus.New()
var logger *logrus.Entry

var (
	configFilename *string
	debug          *bool
	counter        *int
	version        string
)

func init() {
	configFilename = flag.String("config", "", "Config file")
	debug = flag.Bool("debug", false, "Show debug info")
	counter = flag.Int("counter", 0, "Display messages per second")
	versionFlag := flag.Bool("version", false, "Print version info")

	flag.Parse()

	if *versionFlag {
		displayVersion()
		os.Exit(0)
	}

	if len(*configFilename) == 0 {
		fmt.Println("No config file provided")
		flag.Usage()
		os.Exit(0)
	}

	Logger.Formatter = new(prefixed.TextFormatter)

	// Show debug info if required
	if *debug {
		Logger.Level = logrus.DebugLevel
	}

	if *debug {
		go func() {
			Logger.Debugln(http.ListenAndServe("localhost:6060", nil))
		}()
	}
}

func main() {
	var components []interface{}

	// Initialize logger
	logger = Logger.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	// Initialize forwarder and components
	f := rbforwarder.NewRBForwarder(loadForwarderConfig())
	components = append(components, &limiter.Limiter{Config: loadLimiterConfig()})
	components = append(components, &batcher.Batcher{Config: loadBatchConfig()})
	components = append(components, &httpsender.HTTPSender{Config: loadHTTPConfig()})
	f.PushComponents(components)

	// Initialize kafka
	kafka := &KafkaConsumer{Config: loadKafkaConfig()}

	// Set the forwarder on the kafka consumer
	kafka.forwarder = f

	// Wait for ctrl-c to close the consumer
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)
	go func() {
		<-ctrlc
		kafka.Close()
	}()

	// Start getting messages
	f.Run()
	kafka.Start()
}

func displayVersion() {
	fmt.Println("K2HTTP VERSION:\t\t", version)
	fmt.Println("RBFORWARDER VERSION:\t", rbforwarder.Version)
}

func loadConfig(filename, component string) (config map[string]interface{}, err error) {
	generalConfig := make(map[string]interface{})
	config = make(map[string]interface{})

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	yaml.Unmarshal([]byte(data), &generalConfig)
	if err != nil {
		return
	}

	for k, v := range generalConfig[component].(map[interface{}]interface{}) {
		config[k.(string)] = v
	}

	return
}

func loadForwarderConfig() rbforwarder.Config {
	pipelineConfig, err := loadConfig(*configFilename, "pipeline")
	if err != nil {
		logger.Fatal(err)
	}

	config := rbforwarder.Config{}
	if retries, ok := pipelineConfig["retries"].(int); ok {
		config.Retries = retries
	} else {
		logger.Fatal("Invalid 'retries' option")
	}
	if backoff, ok := pipelineConfig["backoff"].(int); ok {
		config.Backoff = backoff
	} else {
		logger.Fatal("Invalid 'backoff' option")
	}
	if queue, ok := pipelineConfig["queue"].(int); ok {
		config.QueueSize = queue
	} else {
		logger.Fatal("Invalid 'queue' option")
	}

	logger.WithFields(map[string]interface{}{
		"retries": config.Retries,
		"backoff": config.Backoff,
		"queue":   config.QueueSize,
	}).Info("Forwarder config")

	return config
}

func loadBatchConfig() batcher.Config {
	batchConfig, err := loadConfig(*configFilename, "batch")
	if err != nil {
		logger.Fatal(err)
	}

	config := batcher.Config{}
	if workers, ok := batchConfig["workers"].(int); ok {
		config.Workers = workers
	} else {
		config.Workers = 1
	}
	if TimeoutMillis, ok := batchConfig["timeoutMillis"].(int); ok {
		config.TimeoutMillis = uint(TimeoutMillis)
	} else {
		logger.Fatal("Invalid 'TimeoutMillis' option")
	}
	if size, ok := batchConfig["size"].(int); ok {
		config.Limit = uint64(size)
	} else {
		logger.Fatal("Invalid 'size' option")
	}
	if deflate, ok := batchConfig["deflate"].(bool); ok {
		config.Deflate = deflate
	}

	logger.WithFields(map[string]interface{}{
		"workers":       config.Workers,
		"timeoutMillis": config.TimeoutMillis,
		"size":          config.Limit,
		"deflate":       config.Deflate,
	}).Info("Batch config")

	return config
}

func loadHTTPConfig() httpsender.Config {
	httpConfig, err := loadConfig(*configFilename, "http")
	if err != nil {
		logger.Fatal(err)
	}

	config := httpsender.Config{}
	if workers, ok := httpConfig["workers"].(int); ok {
		config.Workers = workers
	} else {
		config.Workers = 1
	}
	if *debug {
		config.Logger = logger.WithField("prefix", "http sender")
		config.Debug = true
	}
	if url, ok := httpConfig["url"].(string); ok {
		config.URL = url
	} else {
		logger.Fatal("Invalid 'url' option")
	}

	if insecure, ok := httpConfig["insecure"].(bool); ok {
		config.Insecure = insecure
	}

	logger.WithFields(map[string]interface{}{
		"workers": config.Workers,
		"debug":   config.Debug,
		"url":     config.URL,
		"insecure": config.insecure,
	}).Info("HTTP config")

	return config
}

func loadLimiterConfig() limiter.Config {
	limiterConfig, err := loadConfig(*configFilename, "limiter")
	if err != nil {
		logger.Fatal(err)
	}

	config := limiter.Config{}
	if MessageLimit, ok := limiterConfig["max_messages"].(int); ok {
		config.MessageLimit = uint64(MessageLimit)
	} else {
		config.MessageLimit = 0
	}
	if BytesLimit, ok := limiterConfig["max_bytes"].(int); ok {
		if config.MessageLimit > 0 {
			config.MessageLimit = 0
			logger.Warning("Ignoring \"max_messages\" option")
		}
		config.BytesLimit = uint64(BytesLimit)
	} else {
		config.BytesLimit = 0
	}

	logger.WithFields(map[string]interface{}{
		"max_messages": config.MessageLimit,
		"max_bytes":    config.BytesLimit,
	}).Info("Limiter config")

	return config
}

func loadKafkaConfig() KafkaConfig {
	kafkaConfig, err := loadConfig(*configFilename, "kafka")
	if err != nil {
		logger.Fatal(err)
	}

	config := KafkaConfig{}

	batchConfig, err := loadConfig(*configFilename, "batch")
	config.consumerGroupConfig = cluster.NewConfig()
	if err == nil {
		if deflate, ok := batchConfig["deflate"].(bool); ok {
			config.deflate = deflate
		}
	}
	if *debug {
		sarama.Logger = logger.WithField("prefix", "kafka-consumer")
	}
	if consumerGroup, ok := kafkaConfig["consumergroup"].(string); ok {
		config.consumergroup = consumerGroup
		config.consumerGroupConfig.ClientID = consumerGroup
	} else {
		config.consumergroup = "k2http"
		config.consumerGroupConfig.ClientID = "k2http"
	}
	if broker, ok := kafkaConfig["broker"].(string); ok {
		config.brokers = strings.Split(broker, ",")
	} else {
		logger.Fatal("Invalid 'broker' option")
	}
	if topics, ok := kafkaConfig["topics"].([]interface{}); ok {
		for _, topic := range topics {
			config.topics = append(config.topics, topic.(string))
		}
	}

	config.consumerGroupConfig.Config.Consumer.Offsets.CommitInterval = 1 * time.Second
	config.consumerGroupConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	// config.consumerGroupConfig.Consumer.MaxProcessingTime = 5 * time.Second

	return config
}
