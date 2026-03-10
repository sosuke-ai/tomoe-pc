package speaker

import (
	"fmt"
	"sync"
)

// DefaultThreshold is the default cosine similarity threshold for same-speaker assignment.
const DefaultThreshold = 0.65

// Tracker performs online speaker clustering using cosine similarity of embeddings.
// Speakers are labeled "Person 1", "Person 2", etc.
type Tracker struct {
	mu        sync.Mutex
	threshold float64
	centroids [][]float32 // one centroid per known speaker
	counts    []int       // number of embeddings merged into each centroid
}

// NewTracker creates a Tracker with the given cosine similarity threshold.
// Embeddings with similarity >= threshold to a centroid are assigned to that speaker.
func NewTracker(threshold float64) *Tracker {
	if threshold <= 0 || threshold > 1 {
		threshold = DefaultThreshold
	}
	return &Tracker{
		threshold: threshold,
	}
}

// Assign assigns an embedding to a speaker, creating a new speaker if no match is found.
// Returns a label like "Person 1", "Person 2", etc.
func (t *Tracker) Assign(embedding []float32) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(embedding) == 0 {
		return "Unknown"
	}

	// Find the best matching centroid
	bestIdx := -1
	bestSim := 0.0

	for i, centroid := range t.centroids {
		sim := CosineSimilarity(embedding, centroid)
		if sim > bestSim {
			bestSim = sim
			bestIdx = i
		}
	}

	if bestIdx >= 0 && bestSim >= t.threshold {
		// Update centroid with running average
		t.updateCentroid(bestIdx, embedding)
		return fmt.Sprintf("Person %d", bestIdx+1)
	}

	// New speaker
	newCentroid := make([]float32, len(embedding))
	copy(newCentroid, embedding)
	t.centroids = append(t.centroids, newCentroid)
	t.counts = append(t.counts, 1)
	return fmt.Sprintf("Person %d", len(t.centroids))
}

// Reset clears all speaker centroids.
func (t *Tracker) Reset() {
	t.mu.Lock()
	t.centroids = nil
	t.counts = nil
	t.mu.Unlock()
}

// NumSpeakers returns the number of identified speakers.
func (t *Tracker) NumSpeakers() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.centroids)
}

// updateCentroid updates a centroid with a new embedding using running average.
func (t *Tracker) updateCentroid(idx int, embedding []float32) {
	count := float32(t.counts[idx])
	newCount := count + 1

	for i := range t.centroids[idx] {
		t.centroids[idx][i] = (t.centroids[idx][i]*count + embedding[i]) / newCount
	}
	t.counts[idx] = int(newCount)
}
