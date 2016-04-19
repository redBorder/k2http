# K2HTTP2

k2http is a application that forwards messages from kafka to an  HTTP
endpoint using [rbforwarder](https://github.com/redBorder/rbforwarder)
package as backend.

## Installing

To install this application.

1. Clone this repo and cd to the project

    ```bash
    git clone git@gitlab.redborder.lan:core-developers/k2http2.git
    cd k2http2
    ```
2. Install dependencies

    ```bash
    make get
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
backend:
  workers: 5              # Number of workers
  queue: 1000             # Max internal queue size
  retries: -1             # Number of retries on fail (-1 not limited)
  backoff: 30             # Time to wait between retries in seconds             
  max_messages: 6000      # Max messages per second
  #max_bytes: 5242880     # Max bytes per second
  showcounter: 60         # Time in secods to show message ratio

kafka:
  broker: "127.0.0.1:9092"  # Kafka brokers
  consumergroup: "k2http"   # Consumer group ID   
  begining: false           # Reset offset
  topics:                   # Kafka topics to listen
    - rb_nmsp
    - rb_radius
    - rb_flow
    - rb_loc
    - rb_monitor
    - rb_state
    - rb_social

http:
  url: "https://example.com:8080" # URL to the server
  insecure: true                  # Skip SSSL verification
  batchsize: 1000                 # Max messages per batch
  batchtimeout: 100               # Max time to wait for send a batch
  deflate: true                   # Use deflate
  endpoint: topic                 # String to add to the URL as suffix
```
