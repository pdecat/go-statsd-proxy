// runner to start up the proxy
package statsdproxy

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	CHANNEL_SIZE = 100
	WORKER_COUNT = 8
)

// variable to indicate whether or not we run in DebugMode
var DebugMode bool

// channel to gather internal metrics
var internalMetrics chan StatsDMetric
var metricsOutput chan metricsRequest

type StatsDMetric struct {
	name  string
	value float64
	raw   string
}

// StartProxy sets up everything
func StartProxy(cfgFilePath string, quit chan bool) error {
	var err error
	config, err := NewConfig(cfgFilePath)
	if err != nil {
		log.Printf("Error parsing config file: %s (exiting...)", err)
		return nil
	}
	internalMetrics = make(chan StatsDMetric, CHANNEL_SIZE)
	metricsOutput = make(chan metricsRequest, CHANNEL_SIZE)

	go metricsCollector(internalMetrics)
	go StartMainListener(*config)
	go StartManagementConsole(*config)

	// wait until you're told to quit
	<-quit

	return nil
}

// StartMainListener sets up the main UDP listener. Everything that is needed to
// receive and relay metrics
func StartMainListener(config ProxyConfig) error {
	if config.Host == "" {
		config.Host, _ = os.Hostname()
	}

	log.Printf("Starting StatsD listener on %s and port %d", config.Host, config.Port)
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(config.Host), Port: config.Port})
	if err != nil {
		log.Printf("Error setting up listener: %s (exiting...)", err)
		return nil
	}

	relayChannel := make(chan StatsDMetric, CHANNEL_SIZE)
	hashRing := NewHashRing(config.Mirror)
	for _, node := range config.Nodes {
		if node.Host == "" {
			node.Host, _ = os.Hostname()
		}

		backend := NewStatsDBackend(node.Host, node.Port, node.ManagementPort, config.CheckInterval)
		if DebugMode {
			log.Printf("Adding backend %s:%d", backend.Host, backend.Port)
		}
		err = hashRing.Add(backend)
		if err != nil {
			log.Printf("Error adding backend to hash ring: %s", err)
		}
	}
	go relaymetric(hashRing, relayChannel)

	workerChannel := make(chan []byte)

	for x := 0; x < WORKER_COUNT; x++ {
		go handleConnection(workerChannel, relayChannel)
	}

	for {
		buf := make([]byte, 512)
		num, _, err := listener.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP buffer: %s (skipping...)", err)
			return nil
		}

		workerChannel <- buf[0:num]
	}
}

// handleConnection handles the actual incoming connection and figure out which packet types we
// got sent.
// accepts a byte array of data
func handleConnection(dataChannel chan []byte, relayChannel chan StatsDMetric) {
	for {
		select {
		case data := <-dataChannel:
			if DebugMode {
				log.Printf("Got packet: %s", string(data))
			}
			metrics := strings.Split(string(data), "\n")
			for _, str := range metrics {
				metric := parsePacketString(str)
				internalMetrics <- StatsDMetric{name: "packets_received", value: 1}
				relayChannel <- *metric
			}

		}
	}
}

// parsePacketString parses a string into a statsd packet
// accepts a string of data
// returns a StatsDMetric
func parsePacketString(data string) *StatsDMetric {
	ret := new(StatsDMetric)
	first := strings.Split(data, ":")
	if len(first) < 2 {
		log.Printf("Malformatted metric: %s", data)
		return ret
	}
	name := first[0]
	second := strings.Split(first[1], "|")
	value64, _ := strconv.ParseInt(second[0], 10, 0)
	value := float64(value64)
	// check for a samplerate
	third := strings.Split(second[1], "@")
	metricType := third[0]

	switch metricType {
	case "c", "ms", "g", "h", "s":
		ret.name = name
		ret.value = value
		ret.raw = data
	default:
		log.Printf("Unknown metrics type: %s", metricType)
	}

	return ret
}

// relaymetric relays a metric to selected statsd backends
func relaymetric(ring *HashRing, relayChannel chan StatsDMetric) {
	for {
		select {
		case metric := <-relayChannel:
			// find out which backend to relay to and do it
			backends, err := ring.GetBackendsForMetric(metric.name)
			if err != nil {
				log.Printf("Unable to get backend for metric: %s", metric.name)
				return
			}

			for _, backend := range backends {
				if DebugMode {
					log.Printf("relaying metric: %s to %s:%d", metric.raw,
						backend.Host, backend.Port)
				}
				backend.Send(metric.raw)
			}
		}
	}
}
