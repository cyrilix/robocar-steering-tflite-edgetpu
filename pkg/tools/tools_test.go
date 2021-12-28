package tools

import (
	"testing"
)

func Test_LinearBin(t *testing.T) {
	type args struct {
		arr    []byte
		n      int
		offset int
		r      float64
	}
	tests := []struct {
		name  string
		args  args
		want  float64
		want1 float64
	}{
		{
			name: "default",
			args: args{
				arr:    []byte{0, 0, 0, 0, 0, 0, 0, 255, 0, 0, 0, 0, 0, 0, 0},
				n:      15,
				offset: -1,
				r:      2.0,
			},
			want:  0.,
			want1: 1.0,
		},
		{
			name: "left",
			args: args{
				arr:    []byte{255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				n:      15,
				offset: -1,
				r:      2.0,
			},
			want:  -1.,
			want1: 1.0,
		},
		{
			name: "right",
			args: args{
				arr:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255},
				n:      15,
				offset: -1,
				r:      2.0,
			},
			want:  1.,
			want1: 1.0,
		},
		{
			name: "right",
			args: args{
				arr:    []byte{0, 0, 0, 0, 0, 0, 0, 5, 10, 15, 20, 40, 100, 60, 5},
				n:      15,
				offset: -1,
				r:      2.0,
			},
			want:  0.7142857142857142,
			want1: 0.39215686274509803,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := LinearBin(tt.args.arr, tt.args.n, tt.args.offset, tt.args.r)
			if got != tt.want {
				t.Errorf("linearBin() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("linearBin() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
