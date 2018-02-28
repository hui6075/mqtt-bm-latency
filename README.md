MQTT broker latency measure tool
=========
功能：测量MQTT服务器转发时延。 </br>

测试场景：每一个topic都有一个订阅者和一个发布者，订阅者接收到消息时通过时间戳测量消息转发时延。</br>
此外，测量结果还包括发布成功率、发布带宽、转发成功率等。</br>

安装:

```
go get github.com/hui6075/mqtt-bm-latency
```

所有依赖包由[manul](https://github.com/kovetskiy/manul)构建。

A MQTT test tool to measure the broker's forwarding latency.
Scenario: each topic has a subscriber and a publisher, the publisher publishes #count messages and the subscriber would measure the broker's forwarding latency by the timestamp inside each message's payload.
Some other measurement include publish success ratio, publish bandwidth, forwarding success ratio would also be count.

Installation:

```
go get github.com/hui6075/mqtt-bm-latency
```

All dependencies are vendored with [manul](https://github.com/kovetskiy/manul).

The tool supports multiple concurrent clients, configurable message size, etc:
```
> mqtt-bm-latency --help
Usage of ./mqtt-bm-latency:
  -broker string
        MQTT broker endpoint as scheme://host:port (default "tcp://localhost:1883")
  -clients int
        Number of clients pair to start (default 10)
  -count int
        Number of messages to send per pubclient (default 100)
  -format string
        Output format: text|json (default "text")
  -password string
        MQTT password (empty if auth disabled)
  -pubqos int
        QoS for published messages (default 1)
  -quiet
        Suppress logs while running
  -size int
        Size of the messages payload (bytes) (default 100)
  -subqos int
        QoS for subscribed messages (default 1)
  -topic string
        MQTT topic for outgoing messages (default "/test")
  -username string
        MQTT username (empty if auth disabled)
```

Two output formats supported: human-readable plain text and JSON.

Example use and output:

```
> mqtt-bm-latency --broker tcp://127.0.0.1:1883 --count 10 --size 100 --clients 10 --subqos 1 --pubqos 2 --format text
....

=========== PUBLISHER 0 ===========
Publish Success Ratio:   100.000% (10/10)
Runtime (s):             0.035
Pub time min (ms):       1.590
Pub time max (ms):       6.813
Pub time mean (ms):      2.831
Pub time std (ms):       1.594
Pub Bandwidth (msg/sec): 286.345

=========== SUBSCRIBER 4 ===========
Forward Success Ratio:       100.000% (10/10)
Forward latency min (ms):    6.000
Forward latency max (ms):    27.000
Forward latency std (ms):    6.867
Mean forward latency (ms):   15.400

================= TOTAL PUBLISHER (10) =================
Total Publish Success Ratio:   100.000% (100/100)
Total Runtime (sec):           0.037
Average Runtime (sec):         0.031
Pub time min (ms):             1.265
Pub time max (ms):             6.813
Pub time mean mean (ms):       2.462
Pub time mean std (ms):        0.242
Average Bandwidth (msg/sec):   321.516
Total Bandwidth (msg/sec):     3215.158

================= TOTAL SUBSCRIBER (10) =================
Total Forward Success Ratio:      100.000% (100/100)
Forward latency min (ms):         2.000
Forward latency max (ms):         27.000
Forward latency mean std (ms):    4.507
Total Mean forward latency (ms):  10.360

```

Similarly, in JSON:

```
> mqtt-bm-latency --broker tcp://127.0.0.1:1883 --count 10 --size 100 --clients 10 --subqos 1 --pubqos 2 --format json --quiet
{
        "publish runs": [{
                "id": 9,
                "pub_successes": 10,
                "failures": 0,
                "run_time": 0.036987145,
                "pub_time_min": 0.8713970000000001,
                "pub_time_max": 11.59584,
                "pub_time_mean": 2.8900005999999996,
                "pub_time_std": 3.0913799647375466,
                "publish_per_sec": 270.3642035631569
        },			...
	],
        "subscribe runs": [{
                "id": 9,
                "actual_published": 10,
                "received": 10,
                "fwd_success_ratio": 1,
                "fwd_time_min": 10,
                "fwd_time_max": 36,
                "fwd_time_mean": 20.8,
                "fwd_time_std": 9.101892355133874
        },			...
	],
        "publish totals": {
                "publish_success_ratio": 1,
                "successes": 100,
                "failures": 0,
                "total_run_time": 0.039432823000000006,
                "avg_run_time": 0.0363683729,
                "pub_time_min": 0.8713970000000001,
                "pub_time_max": 11.59584,
                "pub_time_mean_avg": 2.5923666000000005,
                "pub_time_mean_std": 0.2653321545301076,
                "total_msgs_per_sec": 2752.9351662099743,
                "avg_msgs_per_sec": 275.2935166209974
        },
        "receive totals": {
                "fwd_success_ratio": 1,
                "successes": 100,
                "actual_total_published": 100,
                "fwd_latency_min": 2,
                "fwd_latency_max": 37,
                "fwd_latency_mean_avg": 14.37,
                "fwd_latency_mean_std": 4.129581630679366
        }
}
```
