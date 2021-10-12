package main

import (
	"flag"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/steering"
	"go.uber.org/zap"
	"log"
	"os"
)

const (
	DefaultClientId = "robocar-steering-tflite-edgetpu"
)

func main() {
	var mqttBroker, username, password, clientId string
	var cameraTopic, steeringTopic string
	var modelPath string
	var edgeVerbosity int

	lgr, err := zap.NewProductionConfig().Build()
	if err != nil {
		log.Fatalf("unable to init logger: %v", err)
	}
	zap.ReplaceGlobals(lgr)

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.StringVar(&modelPath, "model", "", "path to model file")
	flag.StringVar(&steeringTopic, "mqtt-topic-road", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic to publish road detection result, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&cameraTopic, "mqtt-topic-camera", os.Getenv("MQTT_TOPIC_CAMERA"), "Mqtt topic that contains camera frame values, use MQTT_TOPIC_CAMERA if args not set")
	flag.IntVar(&edgeVerbosity, "edge-verbosity", 0, "Edge TPU Verbosity")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if modelPath == "" {
		zap.L().Error("model path is mandatory")
		flag.PrintDefaults()
		os.Exit(1)
	}

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		zap.L().Fatal("unable to connect to mqtt bus", zap.Error(err))
	}
	defer client.Disconnect(50)

	p := steering.NewPart(client, modelPath, steeringTopic, cameraTopic, edgeVerbosity)
	defer p.Stop()

	cli.HandleExit(p)

	err = p.Start()
	if err != nil {
		zap.L().Fatal("unable to start service", zap.Error(err))
	}
}
