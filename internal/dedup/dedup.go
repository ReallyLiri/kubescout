package dedup

import "github.com/adrg/strutil/metrics"

var levenshtein *metrics.Levenshtein

const smallFactor = 1
const bigFactor = 3

func init() {
	levenshtein = metrics.NewLevenshtein()
	levenshtein.CaseSensitive = true
	levenshtein.InsertCost = bigFactor
	levenshtein.DeleteCost = bigFactor
	levenshtein.ReplaceCost = smallFactor
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func AreSimilar(a string, b string, similarityThreshold float64) bool {
	maxLenFactor := bigFactor * max(len(a), len(b))
	if maxLenFactor == 0 {
		return true
	}
	distance := levenshtein.Distance(a, b)
	score := 1 - float64(distance)/float64(maxLenFactor)
	return score >= similarityThreshold
}
