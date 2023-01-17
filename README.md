# RICHSMS

💬 smpp clients' server which is client of smpp server 🤪

## TODO

- [ ] Add security
- [ ] Add database
- [ ] Add get status route
- [ ] Add validations
- [ ] Add documentation

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

### ENV and Clients.json

In root there are examples of files change values for your needs

## Requests & Responses

### Request Create Message

```bash
curl -X POST \
  'HOST:PORT/messages' \
  --header 'Accept: */*' \
  --header 'User-Agent: Thunder Client (https://www.thunderclient.com)' \
  --header 'Content-Type: application/json' \
  --data-raw '{
  "to": "phone_number",
  "text": "otp",
  "api_key": "prefix.somekey"
}
'
```

|Key|Value|Description|Example|
| --- | --- | --- | ---|
|to|phone_number|should be specified phone number with country code started with **+**| +99362235616|
|text|otp|random generated one time password to send specific phone_number| 12345|
|api_key| prefix.somekey| raw prefix to find from json file and check key to authenticate client| pc1.wqeij12o3noqwiedas|

### Response Create Message

```json
{
  "id": "15b0c2f2",
  "otp": "012345",
  "status": 0
}
```

|Key|Value|Description|
| --- | --- | --- |
|id|15b0c2f2|randomly generated id of smpp provider|
|otp|012345|message that we sent with request|
|status| 0| don't understand how our ISP works|

### Request Check Message

```bash
curl -X GET \
  '95.85.108.114:8080/messages?id=15598942' \
  --header 'Accept: */*' \
  --header 'User-Agent: Thunder Client (https://www.thunderclient.com)'
```

|QueryKey|QueryValue|Description|
| --- | --- | --- |
|id|15b0c2f2|randomly generated id of smpp provider|

### Response Check Message

```json
{
  "id": "15b0c2f2",
  "otp": "012345"
}
```

|Key|Value|Description|
| --- | --- | --- |
|id|15b0c2f2|randomly generated id of smpp provider|
|otp|012345|message that we sent with request|
