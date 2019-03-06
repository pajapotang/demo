package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/namsral/flag"
	"github.com/pajapotang/demo/consumer/dep/postgresql"
	"github.com/pajapotang/demo/publisher/dep/elasticsearch"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/snappy"
)

type News struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

var logger = log.With().Str("pkg", "main").Logger()
var (
	kafkaBrokerURL     string
	kafkaVerbose       bool
	kafkaTopic         string
	kafkaConsumerGroup string
	kafkaClientID      string
	news               News
	elasticAddr        string
	elasticIndex       string
	elasticType        string
)

func main() {
	flag.StringVar(&kafkaBrokerURL, "kafka-brokers", "localhost:9092, localhost:19092", "Kafka brokers in comma separated value")
	flag.BoolVar(&kafkaVerbose, "kafka-verbose", true, "Kafka verbose logging")
	flag.StringVar(&kafkaTopic, "kafka-topic", "news.add", "Kafka topic. only one topic per consumer.")
	flag.StringVar(&kafkaConsumerGroup, "kafka-consumer-group", "consumer-group", "Kafka consumer group")
	flag.StringVar(&kafkaClientID, "kafka-client-id", "workerservice", "Kafka client id")
	flag.StringVar(&elasticAddr, "elastic-address", "http://localhost:9200", "Listen address for elastic node")
	flag.StringVar(&elasticIndex, "elastic-index", "news", "a specific index, making it searchable")
	flag.StringVar(&elasticType, "elastic-type", "created", " convenient way to store several types of data in the same index")
	flag.Parse()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	brokers := strings.Split(kafkaBrokerURL, ",")

	config := kafka.ReaderConfig{
		Brokers:         brokers,
		GroupID:         kafkaClientID,
		Topic:           kafkaTopic,
		MinBytes:        10e3,
		MaxBytes:        10e6,
		MaxWait:         1 * time.Second,
		ReadLagInterval: -1,
	}

	reader := kafka.NewReader(config)
	defer reader.Close()

	parent := context.Background()
	elasticClient, err := elasticsearch.Configure(parent, elasticAddr, elasticIndex)

	if err != nil {
		logger.Error().Str("ERROR", err.Error()).Msg("unable to configure elasticsearch")
	}
	defer elasticClient.Stop()

	for {
		snappy.NewCompressionCodec()
		m, err := reader.ReadMessage(parent)

		if err != nil {
			log.Error().Msgf("error while receiving message: %v", err)
			continue
		}

		value := m.Value
		fmt.Printf("message at topic/partiotion/offset %v%v%v: %s\n", m.Topic, m.Partition, m.Offset, string(value))

		json.Unmarshal(value, &news)
		createdLog := postgresql.SaveToDB(news.Author, news.Body)

		res, err := elasticsearch.SendToElastic(parent, createdLog.ID, createdLog.Created, elasticIndex, elasticType)

		if err != nil {
			logger.Error().Str("ERROR", err.Error())
		}

		logger.Info().Msgf("Successfully inserted row of data into %s/%s: %+v", elasticIndex, elasticType, res)
	}
}
