package tools

import (
	"fmt"
	"go.uber.org/zap"
	"sort"
	"strings"
)

type ModelType int

func ParseModelType(s string) ModelType {
	switch strings.ToLower(s) {
	case "categorical":
		return ModelTypeCategorical
	case "linear":
		return ModelTypeLinear
	default:
		return ModelTypeUnknown
	}
}

func (m ModelType) String() string {
	switch m {
	case ModelTypeCategorical:
		return "categorical"
	case ModelTypeLinear:
		return "linear"
	default:
		return "unknown"
	}
}

const (
	ModelTypeUnknown ModelType = iota
	ModelTypeCategorical
	ModelTypeLinear
)

// LinearBin  perform inverse linear_bin, taking
func LinearBin(arr []uint8, n int, offset int, r float64) (float64, float64) {
	outputSize := len(arr)
	type result struct {
		score float64
		index int
	}

	var results []result
	minScore := 0.2
	for i := 0; i < outputSize; i++ {
		score := float64(int(arr[i])) / 255.0
		if score < minScore {
			continue
		}
		results = append(results, result{score: score, index: i})
	}

	if len(results) == 0 {
		zap.L().Warn(fmt.Sprintf("none steering with score > %0.2f found", minScore))
		return 0., 0.
	}
	zap.S().Debugf("raw result: %v", results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	b := results[0].index
	a := float64(b)*(r/(float64(n)+float64(offset))) + float64(offset)
	return a, results[0].score
}
