# RICHSMS

smpp clients' server, which is client of smpp server 🤪

## Prerequirements

1. Install Redis
2. Get smpp authorization

## SoftwareRun

### 1.Redis

We use default values of redis database.
U can customize this values for your needs.

```golang
rdb := redis.NewClient(&redis.Options{
  Addr:     "localhost:6379",
  Password: "", // no password set
  DB:       0,  // use default DB
 })
```

### 2.SMPP

Specify environment variables of your smpp server

```golang
tx := &smpp.Transmitter{
  Addr:   os.Getenv("SMS_IP") + ":" + os.Getenv("SMS_PORT"),
  User:   os.Getenv("SMS_LOGIN"),
  Passwd: os.Getenv("SMS_PASSWORD"),
 }
```

### `.env` and `clients.json`

In root there are examples of files change values for your needs

## Requests & Responses

### Request Create Message

```bash
curl -X POST \
  'http://localhost:8080' \
  --header 'Content-Type: application/json' \
  --data-raw '{
  "to": "+99362235616",
  "text": "hi from milkaxq",
  "api_key": "2e313d17-1ff3-490a-a6bf-b10afbffd9d3"
}'
```

| Key     | Value        | Description                                                            | Example                              |
| ------- | ------------ | ---------------------------------------------------------------------- | ------------------------------------ |
| to      | phone number | should be specified phone number with country code started with **+**  | +99362235616                         |
| text    | message      | your message which your want to send                                   | hi from milkaxq                      |
| api_key | secret key   | raw prefix to find from json file and check key to authenticate client | 2e313d17-1ff3-490a-a6bf-b10afbffd9d3 |

### Response Create Message

```json
{
	"id": "15b0c2f2",
	"message": "hi from milkaxq"
}
```

| Key    | Value    | Description                            |
| ------ | -------- | -------------------------------------- |
| id     | 15b0c2f2 | randomly generated id of smpp provider |
| otp    | 012345   | message that we sent with request      |
| status | 0        | don't understand how our ISP works     |

### Request Check Message

```bash
curl -X GET \
  '95.85.108.114:8080/messages/15b0c2f2' \
  --header 'Accept: */*' \
```

| Param    | Description                            |
| -------- | -------------------------------------- |
| 15b0c2f2 | randomly generated id of smpp provider |

### Response Check Message

```json
{
	"id": "15b0c2f2",
	"message": "hi from milkaxq"
}
```

| Key     | Value           | Description                            |
| ------- | --------------- | -------------------------------------- |
| id      | 15b0c2f2        | randomly generated id of smpp provider |
| message | hi from milkaxq | message that we sent with request      |

## Security

Please, create save api*keys. We recommend to use [uuid v4](<https://en.wikipedia.org/wiki/Universally_unique_identifier#Version_4*(random)>)

## Contributors

-   @milkaxq
-   @richxcame
