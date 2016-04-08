package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/senders/httpsender"
)

const (
	defaultQueueSize = 10000
	defaultWorkers   = 1
	defaultRetries   = 0
)

var (
	configFile *string
	debug      *bool
	logger     *logrus.Logger
)

func init() {
	configFile = flag.String("config", "", "Config file")
	debug = flag.Bool("debug", false, "Show debug info")

	flag.Parse()

	if len(*configFile) == 0 {
		fmt.Println("No config file provided")
		flag.Usage()
		os.Exit(1)
	}

	logger = logrus.New()
}

func main() {

	// Load the configuration from file
	configData, err := LoadConfigFile(*configFile)
	if err != nil {
		logger.Fatal(err)
	}

	// Show debug info if required
	if *debug {
		logger.Level = logrus.DebugLevel
	}

	// Capture ctrl-c
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	// Parse the backend configuration
	rbForwarderConfig := rbforwarder.Config{}

	// Get number of workers
	if workers, ok := configData.Backend["workers"].(int); ok {
		rbForwarderConfig.Workers = workers
	} else {
		rbForwarderConfig.Workers = defaultWorkers
	}

	// Get number of retries per message
	if retries, ok := configData.Backend["retries"].(int); ok {
		rbForwarderConfig.Retries = retries
	} else {
		rbForwarderConfig.Retries = defaultRetries
	}

	// Get queue size
	if queue, ok := configData.Backend["queue"].(int); ok {
		rbForwarderConfig.QueueSize = queue
	} else {
		rbForwarderConfig.QueueSize = defaultQueueSize
	}

	// Get the interval to show message rate
	if interval, ok := configData.Backend["showcounter"].(int); ok {
		rbForwarderConfig.ShowCounter = interval
	}

	// Create forwarder
	forwarder := rbforwarder.NewRBForwarder(rbForwarderConfig)

	// Initialize kafka
	kafka := new(KafkaConsumer)
	kafka.ParseKafkaConfig(configData.Kafka)
	kafka.backend = forwarder

	// Get the HTTP sender helper
	httpSenderHelper := httpsender.NewHelper(configData.HTTP)
	forwarder.SetSenderHelper(httpSenderHelper)

	// Start the backend
	forwarder.Start()

	// Wait for ctrl-c
	<-ctrlc

	defer recoverPanic()
}
