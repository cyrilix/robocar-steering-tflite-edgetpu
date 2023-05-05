package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/metrics"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/oci"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/steering"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/tools"
	"go.uber.org/zap"
	"log"
	"os"
	"regexp"
	"strconv"
)

const (
	DefaultClientId = "robocar-steering-tflite-edgetpu"
)

var (
	modelNameRegex = regexp.MustCompile(".*model_(?P<type>(categorical)|(linear))_(?P<imgWidth>\\d+)x(?P<imgHeight>\\d+)h(?P<horizon>\\d+)_edgetpu.tflite$")
)

func main() {
	var mqttBroker, username, password, clientId string
	var cameraTopic, steeringTopic string
	var modelPath, modelsDir, ociRef string
	var edgeVerbosity int
	var imgWidth, imgHeight, horizon int

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.StringVar(&modelPath, "model", "", "path to model file")
	flag.StringVar(&ociRef, "oci-model", "", "oci image to pull")
	flag.StringVar(&modelsDir, "models-dir", "/tmp/robocar/models", "path where to store model file")
	flag.StringVar(&steeringTopic, "mqtt-topic-road", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic to publish road detection result, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&cameraTopic, "mqtt-topic-camera", os.Getenv("MQTT_TOPIC_CAMERA"), "Mqtt topic that contains camera frame values, use MQTT_TOPIC_CAMERA if args not set")
	flag.IntVar(&edgeVerbosity, "edge-verbosity", 0, "Edge TPU Verbosity")
	flag.IntVar(&imgWidth, "img-width", 0, "image width expected by model")
	flag.IntVar(&imgHeight, "img-height", 0, "image height expected by model")
	flag.IntVar(&horizon, "horizon", 0, "upper zone to crop from image. Models expect size 'imgHeight - horizon'")
	logLevel := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()

	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(*logLevel)
	lgr, err := config.Build()
	if err != nil {
		log.Fatalf("unable to init logger: %v", err)
	}
	defer func() {
		if err := lgr.Sync(); err != nil {
			log.Printf("unable to Sync logger: %v\n", err)
		}
	}()
	zap.ReplaceGlobals(lgr)

	cleanup := metrics.Init(context.Background())
	defer cleanup()
	if modelPath == "" && ociRef == "" {
		zap.L().Error("model path or oci image is mandatory")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if modelPath != "" && ociRef != "" {
		zap.L().Error("model path and oci image are exclusives")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var modelType tools.ModelType
	var width, height, horizonFromName int

	if modelPath != "" {
		modelType, width, height, horizonFromName, err = parseModelName(modelPath)
		if err != nil {
			zap.S().Panicf("bad model name '%v', unable to detect configuration from name pattern: %v", modelPath, err)
		}
	} else {
		modelPath, modelType, width, height, horizonFromName, err = oci.PullOciImage(ociRef, modelsDir)
		if err != nil {
			zap.S().Panicf("bad model name '%v', unable to detect configuration from name pattern: %v", modelPath, err)
		}

	}

	if imgWidth == 0 {
		imgWidth = width
	}
	if imgHeight == 0 {
		imgHeight = height
	}
	if horizonFromName == 0 {
		horizon = horizonFromName
	}
	if imgWidth <= 0 || imgHeight <= 0 {
		zap.L().Error("img-width and img-height are mandatory")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if ociRef == "" {
		zap.S().Infof("model path            : %v", modelPath)
	} else {
		zap.S().Infof("oci image model       : %v", ociRef)
	}
	zap.S().Infof("model type            : %v", modelType)
	zap.S().Infof("model for image width : %v", imgWidth)
	zap.S().Infof("model for image height: %v", imgHeight)
	zap.S().Infof("model with horizon    : %v", horizon)

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		zap.L().Fatal("unable to connect to mqtt bus", zap.Error(err))
	}
	defer client.Disconnect(50)

	p := steering.NewPart(client, modelType, modelPath, steeringTopic, cameraTopic, edgeVerbosity, imgWidth, imgHeight, horizon)
	defer p.Stop()

	cli.HandleExit(p)

	err = p.Start()
	if err != nil {
		zap.L().Fatal("unable to start service", zap.Error(err))
	}
}

func parseModelName(modelPath string) (modelType tools.ModelType, imgWidth, imgHeight int, horizon int, err error) {
	match := modelNameRegex.FindStringSubmatch(modelPath)

	results := map[string]string{}
	for i, name := range match {
		results[modelNameRegex.SubexpNames()[i]] = name
	}
	modelType = tools.ParseModelType(results["type"])
	if modelType == tools.ModelTypeUnknown {
		err = fmt.Errorf("unknown model type '%v'", results["type"])
		return
	}
	imgWidth, err = strconv.Atoi(results["imgWidth"])
	if err != nil {
		err = fmt.Errorf("unable to convert image width '%v' to integer: %v", results["imgWidth"], err)
		return
	}
	imgHeight, err = strconv.Atoi(results["imgHeight"])
	if err != nil {
		err = fmt.Errorf("unable to convert image height '%v' to integer: %v", results["imgHeight"], err)
		return
	}
	horizon, err = strconv.Atoi(results["horizon"])
	if err != nil {
		err = fmt.Errorf("unable to convert horizon '%v' to integer: %v", results["horizon"], err)
		return
	}
	return
}
