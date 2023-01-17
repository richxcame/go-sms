package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
	_ "github.com/joho/godotenv/autoload"
)

type MessageBody struct {
	From     string `json:"from"`
	To       string `json:"to" binding:"required"`
	TextType string `json:"text_type"`
	Text     string `json:"text"`
	APIKey   string `json:"api_key"`
}

func main() {
	// Redis connection
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	s := rdb.Ping(ctx)
	if s.Err() != nil {
		log.Fatalln("Unable to connect to redis, aborting:", s.Err().Error())
	}
	log.Println("Connected to redis server")

	// SMPP connection
	tx := &smpp.Transmitter{
		Addr:   os.Getenv("SMS_IP") + ":" + os.Getenv("SMS_PORT"),
		User:   os.Getenv("SMS_LOGIN"),
		Passwd: os.Getenv("SMS_PASSWORD"),
	}
	conn := tx.Bind()
	var status smpp.ConnStatus
	if status = <-conn; status.Error() != nil {
		log.Fatalln("Unable to connect, aborting:", status.Error())
	}
	log.Println("Connection completed, status:", status.Status().String())

	r := gin.Default()

	// Send the message
	r.POST("/messages", func(c *gin.Context) {
		var body MessageBody
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"id":      "wrong_request_body",
				"message": err.Error(),
			})
			return
		}

		// Prepare clients list
		var clients []string
		clientJSON, _ := os.ReadFile("clients.json")
		err := json.Unmarshal(clientJSON, &clients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"id":      "clients_list_error",
				"message": err.Error(),
			})
			return
		}

		// Check the given "api_key" included in list of clients
		if !Contains(clients, body.APIKey) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"id":      "wrong_api_key",
				"message": "Please, ensure that your api_key is valid",
			})
			return
		}

		if !IsPhone(body.To) {
			c.JSON(http.StatusBadRequest, gin.H{
				"id":      "invalid_phone_number",
				"message": "Phone number must be in format E.164",
			})
			return
		}
		if body.From == "" {
			body.From = os.Getenv("SMS_NUMBER")
		}
		var text pdutext.Codec
		text = pdutext.Raw(body.Text)
		if body.TextType == "GSM7" {
			text = pdutext.GSM7(body.Text)
		} else if body.TextType == "GSM7Packed" {
			text = pdutext.GSM7Packed(body.Text)
		} else if body.TextType == "ISO88595" {
			text = pdutext.ISO88595(body.Text)
		} else if body.TextType == "Latin1" {
			text = pdutext.Latin1(body.Text)
		} else if body.TextType == "UCS2" {
			text = pdutext.UCS2(body.Text)
		}

		// Send message
		sm, err := tx.Submit(&smpp.ShortMessage{
			Src:      body.From,
			Dst:      body.To,
			Text:     text,
			Register: pdufield.NoDeliveryReceipt,
		})

		if err == smpp.ErrNotConnected {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "smpp_connection_error",
				"message": err.Error(),
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "message_send_error",
				"message": err.Error(),
			})
			return
		}

		// Set message and response id to redis
		redisLifeTime, err := strconv.Atoi(os.Getenv("REDIS_LIFE_TIME"))
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "redis_env_error",
				"message": err.Error(),
			})
			return
		}
		err = rdb.SetEx(ctx, sm.RespID(), body.Text, time.Second*time.Duration(redisLifeTime)).Err()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "redis_set_error",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":      sm.RespID(),
			"message": body.Text,
		})
	})

	// Get the sent message
	r.GET("/messages/:id", func(c *gin.Context) {
		id := c.Param("id")
		message, err := rdb.Get(ctx, id).Result()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"id":      "message_not_found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      id,
			"message": message,
		})
	})

	r.Run()

}

// Contains checks the value including in []values
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

// Phone number validation according to E.164
// http://en.wikipedia.org/wiki/E.164
func IsPhone(phone string) bool {
	var pattern = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	match := pattern.FindString(phone)

	return match != ""
}
