package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/senders/httpsender"
	"github.com/x-cray/logrus-prefixed-formatter"
)

const (
	defaultQueueSize = 10000
	defaultWorkers   = 1
	defaultRetries   = 0
	defaultBackoff   = 2
)

var (
	configFile *string
	debug      *bool
	logger     *logrus.Entry

	githash string
	version string
)

func init() {
	configFile = flag.String("config", "", "Config file")
	debug = flag.Bool("debug", false, "Show debug info")
	versionFlag := flag.Bool("version", false, "Print version info")

	flag.Parse()

	if *versionFlag {
		fmt.Println("K2HTTP VERSION:\t\t", version)
		fmt.Println("KHTTP COMMIT:\t\t", githash)
		fmt.Println("RBFORWARDER VERSION:\t", rbforwarder.Version)
		os.Exit(0)
	}

	if len(*configFile) == 0 {
		fmt.Println("No config file provided")
		flag.Usage()
		os.Exit(0)
	}

	log := logrus.New()
	// Show debug info if required
	if *debug {
		log.Level = logrus.DebugLevel
	}
	log.Formatter = new(prefixed.TextFormatter)

	logger = log.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	if *debug {
		go func() {
			log.Infoln(http.ListenAndServe("localhost:6060", nil))
		}()
	}
}

func main() {

	// Load configuration from file
	rbForwarderConfig, kafkaConfig, HTTPConfig, err := LoadConfigFile(*configFile)
	if err != nil {
		logger.Fatal(err)
	}

	// Capture ctrl-c
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	// Create forwarder
	forwarder := rbforwarder.NewRBForwarder(rbForwarderConfig)

	// Initialize kafka
	kafka := new(KafkaConsumer)
	kafka.ParseKafkaConfig(kafkaConfig)
	kafka.backend = forwarder

	// Get the HTTP sender helper
	httpSenderHelper := httpsender.NewHelper(HTTPConfig)
	forwarder.SetSenderHelper(httpSenderHelper)

	// Start the backend
	forwarder.Start()

	// Wait for ctrl-c to close the consumer
	go func() {
		<-ctrlc
		kafka.Close()
		forwarder.Close()
	}()

	// Start getting messages
	kafka.Start()

	defer recoverPanic()
}
