package main

import (
	"github.com/cyrilix/robocar-steering-tflite-edgetpu/pkg/tools"
	"testing"
)

func Test_parseModelName(t *testing.T) {
	type args struct {
		modelPath string
	}
	tests := []struct {
		name          string
		args          args
		wantModelType tools.ModelType
		wantImgWidth  int
		wantImgHeight int
		wantHorizon   int
		wantErr       bool
	}{
		{
			name:          "categorical",
			args:          args{modelPath: "/tmp/model_categorical_120x160h10.tflite"},
			wantModelType: tools.ModelTypeCategorical,
			wantImgWidth:  120,
			wantImgHeight: 160,
			wantHorizon:   10,
			wantErr:       false,
		},
		{
			name:          "linear",
			args:          args{modelPath: "/tmp/model_linear_120x160h10.tflite"},
			wantModelType: tools.ModelTypeLinear,
			wantImgWidth:  120,
			wantImgHeight: 160,
			wantHorizon:   10,
			wantErr:       false,
		},
		{
			name:          "bad-model",
			args:          args{modelPath: "/tmp/model_123_120x160h10.tflite"},
			wantModelType: tools.ModelTypeUnknown,
			wantImgWidth:  0,
			wantImgHeight: 0,
			wantHorizon:   0,
			wantErr:       true,
		},
		{
			name:          "bad-width",
			args:          args{modelPath: "/tmp/model_categorical_ax160h10.tflite"},
			wantModelType: tools.ModelTypeUnknown,
			wantImgWidth:  0,
			wantImgHeight: 0,
			wantHorizon:   0,
			wantErr:       true,
		},
		{
			name:          "bad-height",
			args:          args{modelPath: "/tmp/model_categorical_120xh10.tflite"},
			wantModelType: tools.ModelTypeUnknown,
			wantImgWidth:  0,
			wantImgHeight: 0,
			wantHorizon:   0,
			wantErr:       true,
		},
		{
			name:          "bad-horizon",
			args:          args{modelPath: "/tmp/model_categorical_120x160h.tflite"},
			wantModelType: tools.ModelTypeUnknown,
			wantImgWidth:  0,
			wantImgHeight: 0,
			wantHorizon:   0,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModelType, gotImgWidth, gotImgHeight, gotHorizon, err := parseModelName(tt.args.modelPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseModelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotModelType != tt.wantModelType {
				t.Errorf("parseModelName() gotModelType = %v, want %v", gotModelType, tt.wantModelType)
			}
			if gotImgWidth != tt.wantImgWidth {
				t.Errorf("parseModelName() gotImgWidth = %v, want %v", gotImgWidth, tt.wantImgWidth)
			}
			if gotImgHeight != tt.wantImgHeight {
				t.Errorf("parseModelName() gotImgHeight = %v, want %v", gotImgHeight, tt.wantImgHeight)
			}
			if gotHorizon != tt.wantHorizon {
				t.Errorf("parseModelName() gotHorizon = %v, want %v", gotHorizon, tt.wantHorizon)
			}
		})
	}
}
