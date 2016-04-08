# K2HTTP2

k2http is a application that get messages from kafka and send to an HTTP
endpoint using [rbforwarder](https://github.com/redBorder/rbforwarder) package as backend.

## Example config file

This is an example config file for a MQTT to HTTP with json decoding/encoding:

```yaml
backend:
  workers: 10
  queue: 10000
  retries: 3
kafka:
  broker: "localhost:2181"
  topics:
    - "topic1"
    - "topic2"
http:
  url: "http://localhost:8080"
  batchsize: 100
  batchtimeout: 500
  deflate: true
```
