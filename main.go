package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/GaryBoone/GoStats/stats"
)

// Message describes a message
type Message struct {
	Topic     string
	QoS       byte
	Payload   interface{}
	Sent      time.Time
	Delivered time.Time
	Error     bool
}

// SubResults describes results of a single SUBSCRIBER / run
type SubResults struct {
	ID             int     `json:"id"`
	Published      int64   `json:"actual_published"`
	Received       int64   `json:"received"`
	FwdRatio       float64 `json:"fwd_success_ratio"`
	FwdLatencyMin  float64 `json:"fwd_time_min"`
	FwdLatencyMax  float64 `json:"fwd_time_max"`
	FwdLatencyMean float64 `json:"fwd_time_mean"`
	FwdLatencyStd  float64 `json:"fwd_time_std"`
}

// TotalSubResults describes results of all SUBSCRIBER / runs
type TotalSubResults struct {
	TotalFwdRatio     float64 `json:"fwd_success_ratio"`
	TotalReceived     int64   `json:"successes"`
	TotalPublished    int64   `json:"actual_total_published"`
	FwdLatencyMin     float64 `json:"fwd_latency_min"`
	FwdLatencyMax     float64 `json:"fwd_latency_max"`
	FwdLatencyMeanAvg float64 `json:"fwd_latency_mean_avg"`
	FwdLatencyMeanStd float64 `json:"fwd_latency_mean_std"`
}

// PubResults describes results of a single PUBLISHER / run
type PubResults struct {
	ID          int     `json:"id"`
	Successes   int64   `json:"pub_successes"`
	Failures    int64   `json:"failures"`
	RunTime     float64 `json:"run_time"`
	PubTimeMin  float64 `json:"pub_time_min"`
	PubTimeMax  float64 `json:"pub_time_max"`
	PubTimeMean float64 `json:"pub_time_mean"`
	PubTimeStd  float64 `json:"pub_time_std"`
	PubsPerSec  float64 `json:"publish_per_sec"`
}

// TotalPubResults describes results of all PUBLISHER / runs
type TotalPubResults struct {
	PubRatio        float64 `json:"publish_success_ratio"`
	Successes       int64   `json:"successes"`
	Failures        int64   `json:"failures"`
	TotalRunTime    float64 `json:"total_run_time"`
	AvgRunTime      float64 `json:"avg_run_time"`
	PubTimeMin      float64 `json:"pub_time_min"`
	PubTimeMax      float64 `json:"pub_time_max"`
	PubTimeMeanAvg  float64 `json:"pub_time_mean_avg"`
	PubTimeMeanStd  float64 `json:"pub_time_mean_std"`
	TotalMsgsPerSec float64 `json:"total_msgs_per_sec"`
	AvgMsgsPerSec   float64 `json:"avg_msgs_per_sec"`
}

// JSONResults are used to export results as a JSON document
type JSONResults struct {
	PubRuns   []*PubResults    `json:"publish runs"`
	SubRuns   []*SubResults    `json:"subscribe runs"`
	PubTotals *TotalPubResults `json:"publish totals"`
	SubTotals *TotalSubResults `json:"receive totals"`
}

func main() {

	var (
		broker      = flag.String("broker", "tcp://localhost:1883", "MQTT broker endpoint as scheme://host:port")
		topic       = flag.String("topic", "/test", "MQTT topic for outgoing messages")
		username    = flag.String("username", "", "MQTT username (empty if auth disabled)")
		password    = flag.String("password", "", "MQTT password (empty if auth disabled)")
		pubqos      = flag.Int("pubqos", 1, "QoS for published messages")
		subqos      = flag.Int("subqos", 1, "QoS for subscribed messages")
		size        = flag.Int("size", 100, "Size of the messages payload in bytes")
		count       = flag.Int("count", 100, "Number of messages to send per pubclient")
		clients     = flag.Int("clients", 10, "Number of clients pair to start")
		keepalive   = flag.Int("keepalive", 60, "Keep alive period in seconds")
		format      = flag.String("format", "text", "Output format: text|json")
		quiet       = flag.Bool("quiet", false, "Suppress logs while running (default false)")
		protocolver = flag.Uint("protocolver", 3, "MQTT protocol version: 3 means 3.1, 4 means 3.1.1")
	)

	flag.Parse()
	if *clients < 1 {
		log.Fatal("Invlalid arguments")
	}

	//start subscribe

	subResCh := make(chan *SubResults)
	jobDone := make(chan bool)
	subDone := make(chan bool)
	subCnt := 0

	if !*quiet {
		log.Printf("Starting subscribe..\n")
	}

	for i := 0; i < *clients; i++ {
		sub := &SubClient{
			ID:          i,
			BrokerURL:   *broker,
			BrokerUser:  *username,
			BrokerPass:  *password,
			SubTopic:    *topic + "-" + strconv.Itoa(i),
			SubQoS:      byte(*subqos),
			KeepAlive:   *keepalive,
			Quiet:       *quiet,
			ProtocolVer: *protocolver,
		}
		go sub.run(subResCh, subDone, jobDone)
	}

SUBJOBDONE:
	for {
		select {
		case <-subDone:
			subCnt++
			if subCnt == *clients {
				if !*quiet {
					log.Printf("all subscribe job done.\n")
				}
				break SUBJOBDONE
			}
		}
	}

	//start publish
	if !*quiet {
		log.Printf("Starting publish..\n")
	}
	pubResCh := make(chan *PubResults)
	start := time.Now()
	for i := 0; i < *clients; i++ {
		c := &PubClient{
			ID:          i,
			BrokerURL:   *broker,
			BrokerUser:  *username,
			BrokerPass:  *password,
			PubTopic:    *topic + "-" + strconv.Itoa(i),
			MsgSize:     *size,
			MsgCount:    *count,
			PubQoS:      byte(*pubqos),
			KeepAlive:   *keepalive,
			Quiet:       *quiet,
			ProtocolVer: *protocolver,
		}
		go c.run(pubResCh)
	}

	// collect the publish results
	pubresults := make([]*PubResults, *clients)
	for i := 0; i < *clients; i++ {
		pubresults[i] = <-pubResCh
	}
	totalTime := time.Now().Sub(start)
	pubtotals := calculatePublishResults(pubresults, totalTime)

	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Second)
		if !*quiet {
			log.Printf("Benchmark will stop after %v seconds.\n", 3-i)
		}
	}

	// notify subscriber that job done
	for i := 0; i < *clients; i++ {
		jobDone <- true
	}

	// collect subscribe results
	subresults := make([]*SubResults, *clients)
	for i := 0; i < *clients; i++ {
		subresults[i] = <-subResCh
	}

	// collect the sub results
	subtotals := calculateSubscribeResults(subresults, pubresults)

	// print stats
	printResults(pubresults, pubtotals, subresults, subtotals, *format)

	if !*quiet {
		log.Printf("All jobs done.\n")
	}
}

func calculatePublishResults(pubresults []*PubResults, totalTime time.Duration) *TotalPubResults {
	pubtotals := new(TotalPubResults)
	pubtotals.TotalRunTime = totalTime.Seconds()

	pubTimeMeans := make([]float64, len(pubresults))
	msgsPerSecs := make([]float64, len(pubresults))
	runTimes := make([]float64, len(pubresults))
	bws := make([]float64, len(pubresults))

	pubtotals.PubTimeMin = pubresults[0].PubTimeMin
	for i, res := range pubresults {
		pubtotals.Successes += res.Successes
		pubtotals.Failures += res.Failures
		pubtotals.TotalMsgsPerSec += res.PubsPerSec

		if res.PubTimeMin < pubtotals.PubTimeMin {
			pubtotals.PubTimeMin = res.PubTimeMin
		}

		if res.PubTimeMax > pubtotals.PubTimeMax {
			pubtotals.PubTimeMax = res.PubTimeMax
		}

		pubTimeMeans[i] = res.PubTimeMean
		msgsPerSecs[i] = res.PubsPerSec
		runTimes[i] = res.RunTime
		bws[i] = res.PubsPerSec
	}
	pubtotals.PubRatio = float64(pubtotals.Successes) / float64(pubtotals.Successes+pubtotals.Failures)
	pubtotals.AvgMsgsPerSec = stats.StatsMean(msgsPerSecs)
	pubtotals.AvgRunTime = stats.StatsMean(runTimes)
	pubtotals.PubTimeMeanAvg = stats.StatsMean(pubTimeMeans)
	pubtotals.PubTimeMeanStd = stats.StatsSampleStandardDeviation(pubTimeMeans)

	return pubtotals
}

func calculateSubscribeResults(subresults []*SubResults, pubresults []*PubResults) *TotalSubResults {
	subtotals := new(TotalSubResults)
	fwdLatencyMeans := make([]float64, len(subresults))

	subtotals.FwdLatencyMin = subresults[0].FwdLatencyMin
	for i, res := range subresults {
		subtotals.TotalReceived += res.Received

		if res.FwdLatencyMin < subtotals.FwdLatencyMin {
			subtotals.FwdLatencyMin = res.FwdLatencyMin
		}

		if res.FwdLatencyMax > subtotals.FwdLatencyMax {
			subtotals.FwdLatencyMax = res.FwdLatencyMax
		}

		fwdLatencyMeans[i] = res.FwdLatencyMean
		for _, pubres := range pubresults {
			if pubres.ID == res.ID {
				subtotals.TotalPublished += pubres.Successes
				res.Published = pubres.Successes
				res.FwdRatio = float64(res.Received) / float64(pubres.Successes)
			}
		}
	}
	subtotals.FwdLatencyMeanAvg = stats.StatsMean(fwdLatencyMeans)
	subtotals.FwdLatencyMeanStd = stats.StatsSampleStandardDeviation(fwdLatencyMeans)
	subtotals.TotalFwdRatio = float64(subtotals.TotalReceived) / float64(subtotals.TotalPublished)
	return subtotals
}

func printResults(pubresults []*PubResults, pubtotals *TotalPubResults, subresults []*SubResults, subtotals *TotalSubResults, format string) {
	switch format {
	case "json":
		jr := JSONResults{
			PubRuns:   pubresults,
			SubRuns:   subresults,
			PubTotals: pubtotals,
			SubTotals: subtotals,
		}
		data, _ := json.Marshal(jr)
		var out bytes.Buffer
		json.Indent(&out, data, "", "\t")

		fmt.Println(string(out.Bytes()))
	default:
		fmt.Printf("\n")
		for _, pubres := range pubresults {
			fmt.Printf("=========== PUBLISHER %d ===========\n", pubres.ID)
			fmt.Printf("Publish Success Ratio:   %.3f%% (%d/%d)\n", float64(pubres.Successes)/float64(pubres.Successes+pubres.Failures)*100, pubres.Successes, pubres.Successes+pubres.Failures)
			fmt.Printf("Runtime (s):             %.3f\n", pubres.RunTime)
			fmt.Printf("Pub time min (ms):       %.3f\n", pubres.PubTimeMin)
			fmt.Printf("Pub time max (ms):       %.3f\n", pubres.PubTimeMax)
			fmt.Printf("Pub time mean (ms):      %.3f\n", pubres.PubTimeMean)
			fmt.Printf("Pub time std (ms):       %.3f\n", pubres.PubTimeStd)
			fmt.Printf("Pub Bandwidth (msg/sec): %.3f\n", pubres.PubsPerSec)
		}
		fmt.Printf("\n")
		for _, subres := range subresults {
			fmt.Printf("=========== SUBSCRIBER %d ===========\n", subres.ID)
			fmt.Printf("Forward Success Ratio:       %.3f%% (%d/%d)\n", subres.FwdRatio*100, subres.Received, subres.Published)
			fmt.Printf("Forward latency min (ms):    %.3f\n", subres.FwdLatencyMin)
			fmt.Printf("Forward latency max (ms):    %.3f\n", subres.FwdLatencyMax)
			fmt.Printf("Forward latency std (ms):    %.3f\n", subres.FwdLatencyStd)
			fmt.Printf("Mean forward latency (ms):   %.3f\n", subres.FwdLatencyMean)
		}
		fmt.Printf("\n")
		fmt.Printf("================= TOTAL PUBLISHER (%d) =================\n", len(pubresults))
		fmt.Printf("Total Publish Success Ratio:   %.3f%% (%d/%d)\n", pubtotals.PubRatio*100, pubtotals.Successes, pubtotals.Successes+pubtotals.Failures)
		fmt.Printf("Total Runtime (sec):           %.3f\n", pubtotals.TotalRunTime)
		fmt.Printf("Average Runtime (sec):         %.3f\n", pubtotals.AvgRunTime)
		fmt.Printf("Pub time min (ms):             %.3f\n", pubtotals.PubTimeMin)
		fmt.Printf("Pub time max (ms):             %.3f\n", pubtotals.PubTimeMax)
		fmt.Printf("Pub time mean mean (ms):       %.3f\n", pubtotals.PubTimeMeanAvg)
		fmt.Printf("Pub time mean std (ms):        %.3f\n", pubtotals.PubTimeMeanStd)
		fmt.Printf("Average Bandwidth (msg/sec):   %.3f\n", pubtotals.AvgMsgsPerSec)
		fmt.Printf("Total Bandwidth (msg/sec):     %.3f\n\n", pubtotals.TotalMsgsPerSec)

		fmt.Printf("================= TOTAL SUBSCRIBER (%d) =================\n", len(subresults))
		fmt.Printf("Total Forward Success Ratio:      %.3f%% (%d/%d)\n", subtotals.TotalFwdRatio*100, subtotals.TotalReceived, subtotals.TotalPublished)
		fmt.Printf("Forward latency min (ms):         %.3f\n", subtotals.FwdLatencyMin)
		fmt.Printf("Forward latency max (ms):         %.3f\n", subtotals.FwdLatencyMax)
		fmt.Printf("Forward latency mean std (ms):    %.3f\n", subtotals.FwdLatencyMeanStd)
		fmt.Printf("Total Mean forward latency (ms):  %.3f\n\n", subtotals.FwdLatencyMeanAvg)
	}
	return
}
