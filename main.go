package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
)

type MessageBody struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to" binding:"required"`
	TextType string `json:"text_type"`
	Text     string `json:"text"`
	APIKey   string `json:"api_key"`
}

type Client struct {
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
}

func main() {
	// connect redis
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

	// Connect with smpp server
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

	r.GET("/epoch", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"time": time.Now().Unix(),
		})
	})

	r.POST("/messages", func(c *gin.Context) {
		var (
			body    MessageBody
			clients []Client
		)
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"id":      "wrong_request_body",
				"message": err.Error(),
			})
			return
		}
		// validate api key
		// 1.Split to get prefix
		splittedAPIKey := strings.Split(body.APIKey, ".")
		prefix := splittedAPIKey[0]
		APIKey := splittedAPIKey[1]

		// 2.Read from file of clients
		clientJSON, _ := os.ReadFile("clients.json")
		err := json.Unmarshal(clientJSON, &clients)
		CheckError(err)

		// 3.Find and compare api key
		for _, v := range clients {
			if v.Name == prefix {
				if v.APIKey != APIKey {
					c.JSON(http.StatusBadRequest, gin.H{
						"message": "api_key_error",
					})
					return
				}
			}
		}

		// // 4 Validate phone number
		if body.From == "" {
			body.From = os.Getenv("SMS_NUMBER")
		}
		if body.ID == "" {
			body.ID = uuid.New().String()
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
		// 5. Send Message
		sm, err := tx.Submit(&smpp.ShortMessage{
			Src:      body.From,
			Dst:      body.To,
			Text:     text,
			Register: pdufield.NoDeliveryReceipt,
			TLVFields: pdutlv.Fields{
				pdutlv.TagReceiptedMessageID: pdutlv.CString(body.ID),
			},
		})

		if err == smpp.ErrNotConnected {
			// ? Send warning
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "smpp_connection_error",
				"message": err.Error(),
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "smpp_connection_error",
				"message": err.Error(),
			})
			return
		}
		// 6. Add to redis otp with expiration time
		redisLifeTime, err := strconv.Atoi(os.Getenv("REDIS_LIFE_TIME"))
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "env_error",
				"message": err.Error(),
			})
			return
		}
		err = rdb.SetEx(ctx, sm.RespID(), body.Text, time.Minute*time.Duration(redisLifeTime)).Err()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"id":      "redis_set_error",
				"message": err.Error(),
			})
			return
		}

		c.JSON(201, gin.H{
			"id":     sm.RespID(),
			"status": sm.Resp().Header().Status,
			"otp":    body.Text,
		})
	})

	r.GET("messages", func(c *gin.Context) {
		id := c.Query("id")
		val, err := rdb.Get(ctx, id).Result()
		if err != nil {
			panic(err)
		}
		c.JSON(200, gin.H{
			"id":  id,
			"otp": val,
		})
	})

	r.Run()

}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
