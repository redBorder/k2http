# K2HTTP2

k2http is a application that forwards messages from kafka 0.9+ to an HTTP
endpoint using [rbforwarder](https://github.com/redBorder/rbforwarder)
package.

## Installing

To install this application ensure you have **glide** installed.

```bash
curl https://glide.sh/get | sh
```

And then:

1. Clone this repo and cd to the project

    ```bash
    git clone git@gitlab.redborder.lan:core-developers/k2http2.git
    cd k2http2
    ```
2. Install dependencies

    ```bash
    glide update
    ```
3. Install on desired directory

    ```bash
    prefix=/opt/rb make install
    ```

## Usage

```
Usage of k2http:
  --config string
        Config file
  --debug
        Show debug info
  --version
        Print version info
```

To run `k2http` just execute the following command:

```bash
k2http --config path/to/config/file
```

## Example config file

```yaml
pipeline:
  queue: 1000                     # Max internal queue size
  backoff: 10                     # Time to wait between retries in seconds             
  retries: 3                      # Number of retries on fail (-1 not limited)

limiter:
  max_messages: 5000              # Max messages per second
  # max_bytes: 5242880            # Max bytes per second

kafka:
  broker: "localhost:9092"        # Kafka brokers
  consumergroup: "k2http"         # Consumer group ID   
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
  deflate: false                  # Use deflate to compress batches

http:
  workers: 1
  url: "http://localhost:8888"    # Number of workers, one connection per worker
  insecure: false                 # Skip SSSL verification
```

## Using with Docker

You can use the application with Docker. First you need to compile as usual and then generate the docker image:

```bash
make
docker build -t k2http-docker .
```

You can then use the app inside a docker container:

```bash
docker run --rm k2http-docker --version
```
