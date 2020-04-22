package main

import (
	"fmt"
	"log"
	"time"
	"bytes"
	"strconv"
)

import (
	"github.com/GaryBoone/GoStats/stats"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type PubClient struct {
	ID         int
	BrokerURL  string
	BrokerUser string
	BrokerPass string
	PubTopic   string
	MsgSize    int
	MsgCount   int
	PubQoS     byte
        KeepAlive  int
	Quiet      bool
}

func (c *PubClient) run(res chan *PubResults) {
	newMsgs := make(chan *Message)
	pubMsgs := make(chan *Message)
	doneGen := make(chan bool)
	donePub := make(chan bool)
	runResults := new(PubResults)

	started := time.Now()
	// start generator
	go c.genMessages(newMsgs, doneGen)
	// start publisher
	go c.pubMessages(newMsgs, pubMsgs, doneGen, donePub)

	runResults.ID = c.ID
	times := []float64{}
	for {
		select {
		case m := <-pubMsgs:
			if m.Error {
				log.Printf("PUBLISHER %v ERROR publishing message: %v: at %v\n", c.ID, m.Topic, m.Sent.Unix())
				runResults.Failures++
			} else {
				// log.Printf("Message published: %v: sent: %v delivered: %v flight time: %v\n", m.Topic, m.Sent, m.Delivered, m.Delivered.Sub(m.Sent))
				runResults.Successes++
				times = append(times, m.Delivered.Sub(m.Sent).Seconds()*1000) // in milliseconds
			}
		case <-donePub:
			// calculate results
			duration := time.Now().Sub(started)
			runResults.PubTimeMin = stats.StatsMin(times)
			runResults.PubTimeMax = stats.StatsMax(times)
			runResults.PubTimeMean = stats.StatsMean(times)
			runResults.PubTimeStd = stats.StatsSampleStandardDeviation(times)
			runResults.RunTime = duration.Seconds()
			runResults.PubsPerSec = float64(runResults.Successes) / duration.Seconds()

			// report results and exit
			res <- runResults
			return
		}
	}
}

func (c *PubClient) genMessages(ch chan *Message, done chan bool) {
	for i := 0; i < c.MsgCount; i++ {
		ch <- &Message{
			Topic:   c.PubTopic,
			QoS:     c.PubQoS,
			//Payload: make([]byte, c.MsgSize),
		}
	}
	done <- true
	// log.Printf("PUBLISHER %v is done generating messages\n", c.ID)
	return
}

func (c *PubClient) pubMessages(in, out chan *Message, doneGen, donePub chan bool) {
	onConnected := func(client mqtt.Client) {
		ctr := 0
		for {
			select {
			case m := <-in:
				m.Sent = time.Now()
				m.Payload = bytes.Join([][]byte{[]byte(strconv.FormatInt(m.Sent.UnixNano(), 10)), make([]byte, c.MsgSize)}, []byte("#@#"))
				token := client.Publish(m.Topic, m.QoS, false, m.Payload)
				token.Wait()
				if token.Error() != nil {
					log.Printf("PUBLISHER %v Error sending message: %v\n", c.ID, token.Error())
					m.Error = true
				} else {
					m.Delivered = time.Now()
					m.Error = false
				}
				out <- m
				ctr++
			case <-doneGen:
				if !c.Quiet {
					log.Printf("PUBLISHER %v had connected to the broker %v and done publishing for topic: %v\n", c.ID, c.BrokerURL, c.PubTopic)
				}
				donePub <- true
				client.Disconnect(250)
				return
			}
		}
	}

	ka, _ := time.ParseDuration(strconv.Itoa(c.KeepAlive) + "s")

	opts := mqtt.NewClientOptions().
		AddBroker(c.BrokerURL).
		SetClientID(fmt.Sprintf("mqtt-benchmark-%v-%v", time.Now(), c.ID)).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(onConnected).
		SetKeepAlive(ka).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
		log.Printf("PUBLISHER %v lost connection to the broker: %v. Will reconnect...\n", c.ID, reason.Error())
	})
	if c.BrokerUser != "" && c.BrokerPass != "" {
		opts.SetUsername(c.BrokerUser)
		opts.SetPassword(c.BrokerPass)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		log.Printf("PUBLISHER %v had error connecting to the broker: %v\n", c.ID, token.Error())
	}
}
