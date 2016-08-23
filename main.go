package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/components/batch"
	"github.com/redBorder/rbforwarder/components/httpsender"
	"github.com/x-cray/logrus-prefixed-formatter"
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
	version        string
)

func init() {
	configFilename = flag.String("config", "", "Config file")
	debug = flag.Bool("debug", false, "Show debug info")
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
	var workers []int

	// Init logger
	logger = Logger.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	// Capture ctrl-c
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	// Load pipeline config
	pipelineConfig, err := loadConfig(*configFilename, "pipeline")
	if err != nil {
		logger.Fatal(err)
	}

	// Init forwarder
	f := rbforwarder.NewRBForwarder(rbforwarder.Config{
		Retries:   pipelineConfig["retries"].(int),
		Backoff:   pipelineConfig["backoff"].(int),
		QueueSize: pipelineConfig["size"].(int),
	})

	// Load Batch config
	batchConfig, err := loadConfig(*configFilename, "batch")
	if err != nil {
		logger.Fatal(err)
	}

	batch := &batcher.Batcher{
		Config: batcher.Config{
			TimeoutMillis: uint(batchConfig["timeoutMillis"].(int)),
			Limit:         uint64(batchConfig["max_messags"].(int)),
			Deflate:       batchConfig["deflate"].(bool),
		},
	}
	components = append(components, batch)
	workers = append(workers, batchConfig["workers"].(int))

	// Load HTTP config
	httpConfig, err := loadConfig(*configFilename, "http")
	if err != nil {
		logger.Fatal(err)
	}

	sender := &httpsender.HTTPSender{
		URL: httpConfig["url"].(string),
	}
	components = append(components, sender)
	workers = append(workers, httpConfig["workers"].(int))

	// Add componentes to the pipeline
	f.PushComponents(components, workers)

	// Initialize kafka
	kafkaConfig, err := loadConfig(*configFilename, "kafka")
	if err != nil {
		logger.Fatal(err)
	}
	kafka := new(KafkaConsumer)
	kafka.ParseKafkaConfig(kafkaConfig)
	kafka.forwarder = f

	// Wait for ctrl-c to close the consumer
	go func() {
		<-ctrlc
		logger.Info("Terminating...")
		f.Close()
		kafka.Close()
	}()

	// Start getting messages
	f.Run()
	kafka.Start()
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

func displayVersion() {
	fmt.Println("K2HTTP VERSION:\t\t", version)
	fmt.Println("RBFORWARDER VERSION:\t", rbforwarder.Version)
}
