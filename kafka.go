package main

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/redBorder/rbforwarder"
	"github.com/wvanbergen/kafka/consumergroup"
	"github.com/wvanbergen/kazoo-go"
)

// KafkaConsumer get messages from multiple kafka topics
type KafkaConsumer struct {
	backend       *rbforwarder.RBForwarder     // The backend to send messages
	consumerGroup *consumergroup.ConsumerGroup // The main kafka consumer
	Config        KafkaConfig                  // Cofiguration after the parsing
}

// KafkaConfig stores the configuration for the Kafka source
type KafkaConfig struct {
	topics              []string // Topics where listen for messages
	brokers             []string // Brokers to connect
	consumergroup       string   // ID for the consumer
	consumerGroupConfig *consumergroup.Config
}

// Start starts reading messages from kafka and pushing them to the pipeline
func (k *KafkaConsumer) Start() {
	var err error

	// Join the consumer group
	k.consumerGroup, err = consumergroup.JoinConsumerGroup(
		k.Config.consumergroup,
		k.Config.topics,
		k.Config.brokers,
		k.Config.consumerGroupConfig,
	)
	if err != nil {
		logger.Fatal(err)
	}

	// Check for errors
	go func() {
		for err := range k.consumerGroup.Errors() {
			logger.Error(err)
		}
	}()

	var offset uint64
	var eventsReported uint64
	var eventsSent uint64

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
			} else {
				logger.
					WithField("ID", report.ID).
					WithField("STATUS", report.Status).
					WithField("OFFSET", message.Offset).
					Debugf("REPORT")
			}

			if err := k.consumerGroup.CommitUpto(message); err != nil {
				logger.Fatal(err)
			}
			offset++
			eventsReported++
		}

		done <- struct{}{}
	}()

	// Start consuming messages
	for message := range k.consumerGroup.Messages() {
		msg, err := k.backend.TakeMessage()
		if err != nil {
			break
		}

		if _, err := msg.InputBuffer.Write(message.Value); err != nil {
			logger.Error(err)
		}

		msg.Metadata["sarama_message"] = message
		msg.Metadata["topic"] = message.Topic
		if err := msg.Produce(); err != nil {
			break
		}

		eventsSent++
	}

	logger.Info("Consumer terminated")

	<-done
	logger.Infof("TOTAL SENT MESSAGES: %d", eventsSent)
	logger.Infof("TOTAL REPORTS: %d", eventsReported)

	return
}

// Close closes the connection with Kafka
func (k *KafkaConsumer) Close() {
	if err := k.consumerGroup.Close(); err != nil {
		logger.Errorf("Error closing the consumer: %s", err)
	}
}

// ParseKafkaConfig reads the configuration from the YAML config file and store it
// on the instance
func (k *KafkaConsumer) ParseKafkaConfig(config map[string]interface{}) {

	// Create the config
	k.Config.consumerGroupConfig = consumergroup.NewConfig()

	k.Config.consumerGroupConfig.Offsets.ProcessingTimeout = 5 * time.Second

	// Parse the brokers addresses
	if val, ok := config["begining"].(bool); ok {
		k.Config.consumerGroupConfig.Offsets.ResetOffsets = val
	}

	// Parse the brokers addresses
	if config["broker"] != nil {
		k.Config.brokers, k.Config.consumerGroupConfig.Zookeeper.Chroot =
			kazoo.ParseConnectionString(config["broker"].(string))
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
