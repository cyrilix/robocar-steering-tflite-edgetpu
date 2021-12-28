package tools

import (
	"fmt"
	"go.uber.org/zap"
	"sort"
)

// LinearBin  perform inverse linear_bin, taking
func LinearBin(arr []byte, n int, offset int, r float64) (float64, float64) {
	outputSize := len(arr)
	type result struct {
		score float64
		index int
	}

	var results []result
	minScore := 0.2
	for i := 0; i < outputSize; i++ {
		score := float64(arr[i]) / 255.0
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
