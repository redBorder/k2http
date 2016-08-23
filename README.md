# K2HTTP2

k2http is a application that forwards messages from kafka 0.9+ to an HTTP
endpoint using [rbforwarder](https://github.com/redBorder/rbforwarder)
package as backend.

## Installing

To install this application ensure you have **glide** installed.

1. Clone this repo and cd to the project

    ```bash
    git clone git@gitlab.redborder.lan:core-developers/k2http2.git
    cd k2http2
    ```
2. Install dependencies

    ```bash
    glide install
    ```
3. Install on desired directory

    ```bash
    prefix=/opt/rb make install
    ```

## Usage

```
Usage of k2http:
  -config string
        Config file
  -debug
        Show debug info
  -version
        Print version info
```

To run `k2http` just execute the following command:

```bash
k2http --config path/to/config/file
```

## Example config file

```yaml
pipeline:
  backoff: 30                     # Time to wait between retries in seconds             
  queue: 1000                     # Max internal queue size
  retries: -1                     # Number of retries on fail (-1 not limited)

limiter:
  max_messages: 6000              # Max messages per second
  # max_bytes: 5242880            # Max bytes per second

kafka:
  broker: "127.0.0.1:9092"        # Kafka brokers
  consumergroup: "k2http"         # Consumer group ID   
  begining: false                 # Reset offset
  topics:                         # Kafka topics to listen
  - rb_nmsp
  - rb_radius
  - rb_flow
  - rb_loc
  - rb_monitor
  - rb_state
  - rb_social

batch:      
  workers: 1                      # Number of workers
  size: 1000                      # Max messages per batch
  timeoutMillis: 100              # Max time to wait for send a batch
  deflate: true

http:
  workers: 1
  url: "http://localhost:8888"    # Number of workers, one connection per worker
  insecure: true                  # Skip SSSL verification
  endpoint: topic                 # String to append to URL
```
