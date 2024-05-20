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
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/redBorder/rbforwarder"
	cluster "github.com/redBorder/sarama-cluster"
	"github.com/sirupsen/logrus"
)

// KafkaConfig stores the configuration for the Kafka source
type KafkaConfig struct {
	topics              []string // Topics where listen for messages
	brokers             []string // Brokers to connect
	consumergroup       string   // ID for the consumer
	consumerGroupConfig *cluster.Config
	deflate             bool
}

// KafkaConsumer get messages from multiple kafka topics
type KafkaConsumer struct {
	forwarder *rbforwarder.RBForwarder // The backend to send messages
	consumer  *cluster.Consumer
	closed    chan struct{}
	reset     chan struct{}
	Config    KafkaConfig // Cofiguration after the parsing
	wg        sync.WaitGroup
}

// Start starts reading messages from kafka and pushing them to the pipeline
func (k *KafkaConsumer) Start() {
	var eventsReported uint64
	var eventsSent uint64
	var messages uint32
	var err error

	k.closed = make(chan struct{})
	k.reset = make(chan struct{})

	logger = Logger.WithFields(logrus.Fields{
		"prefix": "k2http",
	})

	if *counter > 0 {
		go func() {
			for {
				<-time.After(time.Duration(*counter) * time.Second)
				logger.Infof("Messages per second: %d", atomic.LoadUint32(&messages)/uint32(*counter))
				atomic.StoreUint32(&messages, 0)
			}
		}()
	}

	// Start processing reports
	k.wg.Add(1)
	go func() {
		for r := range k.forwarder.GetOrderedReports() {
			report := r.(rbforwarder.Report)
			message := report.Opaque.(*sarama.ConsumerMessage)

			if report.Code != 0 {
				logger.
					WithField("STATUS", report.Status).
					WithField("OFFSET", message.Offset).
					Error("REPORT")
			}

			k.consumer.MarkOffset(message, "")
			eventsReported++
			atomic.AddUint32(&messages, 1)
		}

		k.wg.Done()
	}()

	// Init consumer, consume errors & messages
consumerLoop:
	for {
		k.consumer, err = cluster.NewConsumer(
			k.Config.brokers,
			k.Config.consumergroup,
			k.Config.topics,
			k.Config.consumerGroupConfig,
		)
		k.Config.consumerGroupConfig.Consumer.Return.Errors = true

		if err != nil {
			logger.Fatal("Failed to start consumer: ", err)
			break
		}

		logger.
			WithField("brokers", k.Config.brokers).
			WithField("consumergroup", k.Config.consumergroup).
			WithField("topics", k.Config.topics).
			Info("Started consumer")

		go func() {
			for err := range k.consumer.Errors() {
				if strings.Contains(err.Error(), "offset is outside the range") {
					logger.Infof("Resetting offsets due to out of range error")
					k.reset <- struct{}{}
					return
				}
			}
		}()

		for {
			select {
			case <-k.closed:
				break consumerLoop
			case <-k.reset:
				logger.Infof("Reset signal received, closing consumer to reset offsets")
				if err := k.consumer.Close(); err != nil {
					logger.Error("Failed to close consumer: ", err)
				}
				k.Config.consumerGroupConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
				continue consumerLoop
			case message := <-k.consumer.Messages():
				if message == nil {
					k.Config.consumerGroupConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
					if err := k.consumer.Close(); err != nil {
						logger.Error("Failed to close consumer: ", err)
					}
					logger.Error("Message is nill")
					break
				}

				opts := map[string]interface{}{
					"http_endpoint": message.Topic,
					"batch_group":   message.Topic,
				}

				if k.Config.deflate {
					opts["http_headers"] = map[string]string{
						"Content-Encoding": "deflate",
					}
				}

				k.forwarder.Produce(message.Value, opts, message)

				eventsSent++
			}
		}
	}

	k.wg.Wait()
	logger.Infof("TOTAL SENT MESSAGES: %d", eventsSent)
	logger.Infof("TOTAL REPORTS: %d", eventsReported)

	return
}

// Close closes the connection with Kafka
func (k *KafkaConsumer) Close() {
	logger.Info("Terminating... Press ctrl+c again to force exit")
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)
	go func() {
		<-ctrlc
		logger.Fatal("Forced exit")
	}()

	k.closed <- struct{}{}
	if err := k.consumer.Close(); err != nil {
		logger.Println("Failed to close consumer: ", err)
	} else {
		logger.Info("Consumer terminated")
	}

	<-time.After(5 * time.Second)
	k.forwarder.Close()
}
