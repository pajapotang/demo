package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestPushOK(t *testing.T) {
	kafkaBrokerURL := "0.0.0.0:9092"
	kafkaClientID := "workerservice"
	kafkaTopic := "test"
	parent := context.Background()
	message := "success"
	value, _ := json.Marshal(message)

	kafkaProducer, err := Configure(strings.Split(kafkaBrokerURL, ","), kafkaClientID, kafkaTopic)
	if err != nil {
		t.Errorf("unable to configure kafka %v", err.Error())
	}

	Push(parent, nil, value)

	kafkaProducer.Close()

	partition := 0

	conn, _ := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", kafkaTopic, partition)

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	batch := conn.ReadBatch(10e3, 1e6)

	b := make([]byte, 10e3)
	for {
		_, err := batch.Read(b)
		if err != nil {
			break
		}
		fmt.Println(string(b))

		if string(b) != message {
			t.Errorf("sender value %s is not match with receiver value %s", message, string(b))
		}
	}

	batch.Close()
	conn.Close()
}
