package search

import (
	"math"
	"strings"
)

// GeneratePseudoEmbedding creates a normalized 384-dim semantic vector from text for local similarity computation
func GeneratePseudoEmbedding(text string, dim int) []float32 {
	if dim <= 0 {
		dim = 384
	}
	vec := make([]float32, dim)
	words := strings.Fields(strings.ToLower(text))

	for _, w := range words {
		var h uint32 = 2166136261
		for i := 0; i < len(w); i++ {
			h ^= uint32(w[i])
			h *= 16777619
		}
		idx := int(h % uint32(dim))
		vec[idx] += 1.0
	}

	// Normalize Vector (L2 norm)
	var sumSquares float64
	for _, v := range vec {
		sumSquares += float64(v * v)
	}
	if sumSquares > 0 {
		norm := float32(math.Sqrt(sumSquares))
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}

// CosineSimilarity computes cosine similarity between two float32 slices
func CosineSimilarity(a, b []float32) float64 {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	if minLen == 0 {
		return 0.0
	}
	var dot, normA, normB float64
	for i := 0; i < minLen; i++ {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
