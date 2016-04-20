package main

import (
	"bytes"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder"
	"gopkg.in/Shopify/sarama.v1"
	"gopkg.in/bsm/sarama-cluster.v2"
)

// KafkaConsumer get messages from multiple kafka topics
type KafkaConsumer struct {
	backend  *rbforwarder.RBForwarder // The backend to send messages
	consumer *cluster.Consumer
	closed   bool
	Config   KafkaConfig // Cofiguration after the parsing
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
	var err error

	var offset uint64
	var eventsReported uint64
	var eventsSent uint64

	logger = Logger.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	// Start processing reports
	done := make(chan struct{})
	go func() {
		for report := range k.backend.GetOrderedReports() {
			message := report.Metadata["sarama_message"].(*sarama.ConsumerMessage)

			if offset != report.ID {
				logger.Fatalf("Unexpected offset. Expected %d, found %d.",
					offset, report.ID)
			}

			if report.StatusCode != 0 {
				logger.
					WithField("ID", report.ID).
					WithField("STATUS", report.Status).
					WithField("OFFSET", message.Offset).
					Errorf("REPORT")
			}

			k.consumer.MarkOffset(message, "")
			offset++
			eventsReported++
		}

		done <- struct{}{}
	}()

	// Init consumer, consume errors & messages
mainLoop:
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

			msg, err := k.backend.TakeMessage()
			if err != nil {
				break mainLoop
			}

			msg.InputBuffer = bytes.NewBuffer(message.Value)
			msg.Metadata["sarama_message"] = message
			msg.Metadata["topic"] = message.Topic
			if err := msg.Produce(); err != nil {
				break mainLoop
			}

			eventsSent++
		}

		if k.closed {
			break
		}
	}

	logger.Info("Consumer terminated")

	<-done
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
	}
}
