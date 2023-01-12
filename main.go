package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
)

type MessageBody struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to" binding:"required"`
	TextType string `json:"text_type"`
	Text     string `json:"text"`
}

func main() {
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
		var body MessageBody
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"id":      "wrong_request_body",
				"message": err.Error(),
			})
			return
		}
		// ? Validate phone number
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

		c.JSON(201, gin.H{
			"id": sm.RespID(),
		})
	})

	r.Run()

}
