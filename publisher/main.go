package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pajapotang/demo/publisher/dep/elasticsearch"
	"github.com/pajapotang/demo/publisher/dep/kafka"
	"github.com/pajapotang/demo/publisher/dep/postgresql"
	"github.com/pajapotang/demo/publisher/util"
	"github.com/rs/zerolog/log"
)

type (
	News struct {
		Author string `json:"author"`
		Body   string `json:"body"`
	}

	NewsList struct {
		ID      int       `json:"id"`
		Author  string    `json:"author"`
		Body    string    `json:"body"`
		Created time.Time `json:"created"`
	}
)

var logger = log.With().Str("pkg", "main").Logger()
var (
	listenAddrAPI  string
	kafkaBrokerURL string
	kafkaVerbose   bool
	kafkaClientID  string
	kafkaTopic     string
	elasticAddr    string
	elasticIndex   string
)

func main() {
	flag.StringVar(&listenAddrAPI, "listen-address", "0.0.0.0:8080", "Listen address for api")
	flag.StringVar(&kafkaBrokerURL, "kafka-brokers", "localhost:9092", "kafka brokers in comma separated value")
	flag.BoolVar(&kafkaVerbose, "kafka-verbose", true, "Kafka verbose logging")
	flag.StringVar(&kafkaClientID, "kafka-client-id", "my-kafka-client", "Kafka client id to connect")
	flag.StringVar(&kafkaTopic, "kafka-topic", "news.add", "Kafka topic to push")
	flag.StringVar(&elasticAddr, "elastic-address", "http://localhost:9200", "Listen address for elastic node")
	flag.StringVar(&elasticIndex, "elastic-index", "news", "a specific index, making it searchable")

	flag.Parse()

	parent := context.Background()
	kafkaProducer, err := kafka.Configure(strings.Split(kafkaBrokerURL, ","), kafkaClientID, kafkaTopic)

	if err != nil {
		logger.Error().Str("ERROR", err.Error()).Msg("unable to configure kafka")
		return
	}
	defer kafkaProducer.Close()
	var errChan = make(chan error, 1)

	elasticClient, err := elasticsearch.Configure(parent, elasticAddr, elasticIndex)

	if err != nil {
		logger.Error().Str("ERROR", err.Error()).Msg("unable to configure elasticsearch")
	}
	defer elasticClient.Stop()

	go func() {
		log.Info().Msgf("starting server at %s", listenAddrAPI)
		errChan <- Server(listenAddrAPI)
	}()

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-signalChan:
		logger.Info().Msg("got an interrupt, exiting...")
	case err := <-errChan:
		if err != nil {
			logger.Error().Err(err).Msg("error while running api, exiting...")
		}
	}
}

func Server(listenAddr string) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.POST("news", PublishDataToKafka)
	router.GET("news", GetNews)

	for _, routeInfo := range router.Routes() {
		logger.Debug().
			Str("path", routeInfo.Path).
			Str("handler", routeInfo.Handler).
			Str("method", routeInfo.Method).
			Msg("registered routes")
	}

	return router.Run(listenAddr)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET")

		c.Next()
	}
}

func PublishDataToKafka(ctx *gin.Context) {
	parent := context.Background()
	defer parent.Done()

	forms := &struct {
		Data []News `form:"data" json:"data"`
	}{}

	ctx.Bind(forms)
	for _, data := range forms.Data {
		formInBytes, err := json.Marshal(data)
		if err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error": map[string]interface{}{
					"message": fmt.Sprint("error while marshalling json: %s", err.Error()),
				},
			})

			ctx.Abort()
			return
		}

		err = kafka.Push(parent, nil, formInBytes)
		if err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"error": map[string]interface{}{
					"message": fmt.Sprintf("error while push message into kafka: %s", err.Error()),
				},
			})

			ctx.Abort()
			return
		}
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "success push data into broker",
	})
}

func GetNews(c *gin.Context) {
	const take = 10
	finishChan := make(chan *postgresql.NewsData)
	url := c.Request.URL
	skip := 0
	var news []NewsList

	page, err := util.GetPage(url)

	if err != nil {
		logger.Error().Str("ERROR", err.Error())
		errorResponse(c, http.StatusBadRequest, err.Error())
	}

	if page > 1 {
		skip = (page - 1) * take
	}

	newsDoc, err := elasticsearch.GetDoc(c.Request.Context(), elasticIndex, skip, take)

	if err != nil {
		logger.Error().Msg(err.Error())
		errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	gophers := len(newsDoc)
	for i := 0; i < gophers; i++ {
		sqlStatement := "select id, author, body, created from news where id=" + strconv.Itoa(int((newsDoc[i].ID)))
		go postgresql.GetNews(finishChan, sqlStatement)
	}

	finishedGophers := 0
	finishLoop := false
	for {
		if finishLoop {
			break
		}
		select {
		case n := <-finishChan:
			if n.Err != nil {
				logger.Error().Msgf("error get data from database: %v", n.Err)
			} else {
				news = append(news, NewsList{
					ID:      int(n.Data.ID),
					Author:  n.Data.Author,
					Body:    n.Data.Body,
					Created: n.Data.Created,
				})
				finishedGophers++
				if finishedGophers == gophers {
					finishLoop = true
				}
			}
		}
	}

	sort.SliceStable(news, func(i, j int) bool {
		return news[i].ID > news[j].ID
	})

	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   news,
	})
}

func errorResponse(c *gin.Context, code int, err string) {
	c.JSON(code, gin.H{
		"error": err,
	})
}
