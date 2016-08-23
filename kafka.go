package main

import (
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	"github.com/bsm/sarama-cluster"
	"github.com/redBorder/rbforwarder"
)

// KafkaConsumer get messages from multiple kafka topics
type KafkaConsumer struct {
	forwarder *rbforwarder.RBForwarder // The backend to send messages
	consumer  *cluster.Consumer
	closed    bool
	Config    KafkaConfig // Cofiguration after the parsing
}

// KafkaConfig stores the configuration for the Kafka source
type KafkaConfig struct {
	topics              []string // Topics where listen for messages
	brokers             []string // Brokers to connect
	consumergroup       string   // ID for the consumer
	consumerGroupConfig *cluster.Config
}

// Start starts reading messages from kafka and pushing them to the pipeline
func (k *KafkaConsumer) Start() {
	var eventsReported uint64
	var eventsSent uint64
	var wg sync.WaitGroup
	var err error

	logger = Logger.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	// Start processing reports
	wg.Add(1)
	go func() {
		for r := range k.forwarder.GetOrderedReports() {
			report := r.(rbforwarder.Report)
			message := report.Opaque.(*sarama.ConsumerMessage)

			if report.Code != 0 {
				logger.
					WithField("STATUS", report.Status).
					WithField("OFFSET", message.Offset).
					Errorf("REPORT")
			}

			k.consumer.MarkOffset(message, "")
			eventsReported++
		}

		wg.Done()
	}()

	// Init consumer, consume errors & messages
	for {
		k.consumer, err = cluster.NewConsumer(
			k.Config.brokers,
			k.Config.consumergroup,
			k.Config.topics,
			k.Config.consumerGroupConfig,
		)
		if err != nil {
			logger.Fatal("Failed to start consumer: ", err)
			break
		}

		logger.
			WithField("brokers", k.Config.brokers).
			WithField("consumergroup", k.Config.consumergroup).
			WithField("topics", k.Config.topics).
			Info("Started consumer")

		// Start consuming messages
		for message := range k.consumer.Messages() {
			if message == nil {
				k.Config.consumerGroupConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
				if err := k.consumer.Close(); err != nil {
					logger.Error("Failed to close consumer: ", err)
				}
				break
			}

			opts := map[string]interface{}{
				"http_endpoint": message.Topic,
				"batch_group":   message.Topic,
			}

			k.forwarder.Produce(message.Value, opts, message)

			eventsSent++
		}

		if k.closed {
			break
		}
	}

	logger.Info("Consumer terminated")

	wg.Wait()
	logger.Infof("TOTAL SENT MESSAGES: %d", eventsSent)
	logger.Infof("TOTAL REPORTS: %d", eventsReported)

	return
}

// Close closes the connection with Kafka
func (k *KafkaConsumer) Close() {
	k.closed = true
	if err := k.consumer.Close(); err != nil {
		logger.Println("Failed to close consumer: ", err)
	}
}

// ParseKafkaConfig reads the configuration from the YAML config file and store it
// on the instance
func (k *KafkaConsumer) ParseKafkaConfig(config map[string]interface{}) {

	// Create the config
	k.Config.consumerGroupConfig = cluster.NewConfig()
	k.Config.consumerGroupConfig.Config.Consumer.Offsets.CommitInterval = 1 * time.Second
	k.Config.consumerGroupConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	k.Config.consumerGroupConfig.Consumer.MaxProcessingTime = 3 * time.Second

	if *debug {
		saramaLogger := Logger.WithField("prefix", "kafka-consumer")
		sarama.Logger = saramaLogger
	}

	// Parse the brokers addresses
	if config["broker"] != nil {
		k.Config.brokers = strings.Split(config["broker"].(string), ",")
	}

	// Parse topics
	if config["topics"] != nil {
		topics := config["topics"].([]interface{})

		for _, topic := range topics {
			k.Config.topics = append(k.Config.topics, topic.(string))
		}
	}

	// Parse consumergroup
	if config["consumergroup"] != nil {
		k.Config.consumergroup = config["consumergroup"].(string)
		k.Config.consumerGroupConfig.ClientID = config["consumergroup"].(string)
	}
}
