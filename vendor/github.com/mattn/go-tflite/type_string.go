// Code generated by "stringer -type TensorType,Status -output type_string.go ."; DO NOT EDIT.

package tflite

import "strconv"

const _TensorType_name = "NoTypeFloat32Int32UInt8Int64StringBoolInt16Complex64Int8"

var _TensorType_index = [...]uint8{0, 6, 13, 18, 23, 28, 34, 38, 43, 52, 56}

func (i TensorType) String() string {
	if i < 0 || i >= TensorType(len(_TensorType_index)-1) {
		return "TensorType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TensorType_name[_TensorType_index[i]:_TensorType_index[i+1]]
}

const _Status_name = "OK"

var _Status_index = [...]uint8{0, 2}

func (i Status) String() string {
	if i < 0 || i >= Status(len(_Status_index)-1) {
		return "Status(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Status_name[_Status_index[i]:_Status_index[i+1]]
}
