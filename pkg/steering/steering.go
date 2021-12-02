package steering

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cyrilix/robocar-base/service"
	"github.com/cyrilix/robocar-protobuf/go/events"
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/metrics"
	"github.com/disintegration/imaging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/proto"
	"github.com/mattn/go-tflite"
	"github.com/mattn/go-tflite/delegates/edgetpu"
	"go.uber.org/zap"
	"image"
	_ "image/jpeg"
	"sort"
	"time"
)

func NewPart(client mqtt.Client, modelPath, steeringTopic, cameraTopic string, edgeVerbosity int, imgWidth, imgHeight, horizon int) *Part {
	return &Part{
		client:        client,
		modelPath:     modelPath,
		steeringTopic: steeringTopic,
		cameraTopic:   cameraTopic,
		edgeVebosity:  edgeVerbosity,
		imgWidth:      imgWidth,
		imgHeight:     imgHeight,
		horizon:       horizon,
	}

}

type Part struct {
	client        mqtt.Client
	steeringTopic string
	cameraTopic   string

	cancel chan interface{}

	options      *tflite.InterpreterOptions
	interpreter  *tflite.Interpreter
	modelPath    string
	model        *tflite.Model
	edgeVebosity int

	imgWidth  int
	imgHeight int
	horizon   int
}

func (p *Part) Start() error {
	p.cancel = make(chan interface{})
	p.model = tflite.NewModelFromFile(p.modelPath)
	if p.model == nil {
		return fmt.Errorf("cannot load model %v", p.modelPath)
	}

	// Get the list of devices
	devices, err := edgetpu.DeviceList()
	if err != nil {
		return fmt.Errorf("could not get EdgeTPU devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no edge TPU devices found")
	}

	// Print the EdgeTPU version
	edgetpuVersion, err := edgetpu.Version()
	if err != nil {
		return fmt.Errorf("cannot get EdgeTPU version: %w", err)
	}
	zap.S().Infof("EdgeTPU Version: %s", edgetpuVersion)
	edgetpu.Verbosity(p.edgeVebosity)

	p.options = tflite.NewInterpreterOptions()
	p.options.SetNumThread(4)
	p.options.SetErrorReporter(func(msg string, userData interface{}) {
		zap.S().Errorw(msg,
			"userData", userData,
		)
	}, nil)

	zap.S().Infof("find %d edgetpu devices", len(devices))
	zap.S().Infow("configure edgetpu",
		"path", devices[0].Path,
		"type", uint32(devices[0].Type),
	)
	// Add the first EdgeTPU device
	d := edgetpu.New(devices[0])
	if d == nil {
		return fmt.Errorf("unable to create new EdgeTpu delegate")
	}
	p.options.AddDelegate(d)

	p.interpreter = tflite.NewInterpreter(p.model, p.options)
	if p.interpreter == nil {
		return fmt.Errorf("cannot create interpreter")
	}

	if err := registerCallbacks(p); err != nil {
		zap.S().Errorw("unable to register callbacks", "error", err)
		return err
	}

	p.cancel = make(chan interface{})
	<-p.cancel
	return nil
}

func (p *Part) Stop() {
	close(p.cancel)
	service.StopService("steering", p.client, p.cameraTopic)
	if p.interpreter != nil {
		p.interpreter.Delete()
	}
	p.interpreter.Delete()
	if p.options != nil {
		p.options.Delete()
	}
	if p.model != nil {
		p.model.Delete()
	}
}

func (p *Part) onFrame(_ mqtt.Client, message mqtt.Message) {
	var msg events.FrameMessage
	err := proto.Unmarshal(message.Payload(), &msg)
	if err != nil {
		zap.S().Errorf("unable to unmarshal protobuf %T message: %v", &msg, err)
		return
	}

	now := time.Now().UnixMilli()
	frameAge := now - msg.Id.CreatedAt.AsTime().UnixMilli()
	go metrics.FrameAge.Record(context.Background(), frameAge)


	img, _, err := image.Decode(bytes.NewReader(msg.GetFrame()))
	if err != nil {
		zap.L().Error("unable to decode frame, skip frame", zap.Error(err))
		return
	}

	steering, confidence, err := p.Value(img)
	inferenceDuration := time.Now().UnixMilli() - now
	go metrics.InferenceDuration.Record(context.Background(), inferenceDuration)

	if err != nil {
		zap.S().Errorw("unable to compute sterring",
			"frame", msg.GetId().GetId(),
			"error", err,
		)
		return
	}
	zap.L().Debug("new steering value",
		zap.Float32("steering", steering),
		zap.Float32("confidence", confidence),
	)
	msgSteering := &events.SteeringMessage{
		Steering:   steering,
		Confidence: confidence,
		FrameRef:   msg.Id,
	}

	payload, err := proto.Marshal(msgSteering)
	if err != nil {
		zap.L().Error("unable to marshal protobuf message", zap.Error(err))
	}
	publish(p.client, p.steeringTopic, payload)
}

func (p *Part) Value(img image.Image) (float32, float32, error) {
	status := p.interpreter.AllocateTensors()
	if status != tflite.OK {
		return 0., 0., fmt.Errorf("tensor allocate failed: %v", status)
	}

	input := p.interpreter.GetInputTensor(0)

	dx := img.Bounds().Dx()
	dy := img.Bounds().Dy()

	if dx != p.imgWidth || dy != p.imgHeight {
		img = imaging.Resize(img, p.imgWidth, p.imgHeight, imaging.NearestNeighbor)
	}
	if p.horizon > 0 {
		img = imaging.Crop(img, image.Rect(0, p.horizon, img.Bounds().Dx(), img.Bounds().Dy()))
	}

	dx = img.Bounds().Dx()
	dy = img.Bounds().Dy()

	bb := make([]byte, dx*dy*3)
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			bb[(y*dx+x)*3+0] = byte(float64(r) / 255.0)
			bb[(y*dx+x)*3+1] = byte(float64(g) / 255.0)
			bb[(y*dx+x)*3+2] = byte(float64(b) / 255.0)
		}
	}
	status = input.CopyFromBuffer(bb)
	if status != tflite.OK {
		return 0., 0., fmt.Errorf("input copy from buffer failed: %v", status)
	}

	status = p.interpreter.Invoke()
	if status != tflite.OK {
		return 0., 0., fmt.Errorf("invoke failed: %v", status)
	}

	output := p.interpreter.GetOutputTensor(0)
	outputSize := output.Dim(output.NumDims() - 1)
	b := make([]byte, outputSize)
	type result struct {
		score float64
		index int
	}
	status = output.CopyToBuffer(&b[0])
	if status != tflite.OK {
		return 0., 0., fmt.Errorf("output failed: %v", status)
	}

	var results []result
	minScore := 0.2
	for i := 0; i < outputSize; i++ {
		score := float64(b[i]) / 255.0
		if score < minScore {
			continue
		}
		results = append(results, result{score: score, index: i})
	}

	if len(results) == 0 {
		zap.L().Warn(fmt.Sprintf("none steering with score > %0.2f found", minScore))
		return 0., 0., nil
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	steering := float64(results[0].index)*(2./float64(outputSize)) - 1
	zap.L().Debug("found steering",
		zap.Float64("steering", steering),
		zap.Float64("score", results[0].score),
	)
	return float32(steering), float32(results[0].score), nil
}

var registerCallbacks = func(p *Part) error {
	err := service.RegisterCallback(p.client, p.cameraTopic, p.onFrame)
	if err != nil {
		return fmt.Errorf("unable to register callback: %w", err)
	}
	return nil
}

var publish = func(client mqtt.Client, topic string, payload []byte) {
	client.Publish(topic, 0, false, payload)
}
