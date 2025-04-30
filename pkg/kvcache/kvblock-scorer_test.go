package kvcache_test

import (
	"testing"

	"github.com/neuralmagic/llm-d-kv-cache-manager/pkg/kvcache"

	"github.com/stretchr/testify/assert"
)

// TestLongestPrefixScorer verifies scoring based on consecutive block hits from the start.
func TestLongestPrefixScorer(t *testing.T) {
	scorer := &kvcache.LongestPrefixScorer{}
	blockKeys := []string{"b1", "b2", "b3", "b4", "b5", "b6"}

	hitmap := map[string][]string{
		"b1": {"pod-a"},
		"b2": {"pod-a"},
		"b3": {"pod-a"},
		"b4": {"pod-b"},
		"b5": {"pod-b"},
		"b6": {"pod-a"},
	}

	expected := map[string]int{
		"pod-a": 3,
		"pod-b": 0,
	}

	scored, err := scorer.Score(blockKeys, hitmap)
	assert.NoError(t, err)
	for pod, score := range scored {
		assert.Equal(t, expected[pod], score)
	}
}

// TestHighestBlockHitScorer verifies scoring based on the highest index where a pod has a block.
func TestHighestBlockHitScorer(t *testing.T) {
	scorer := &kvcache.HighestBlockHitScorer{}
	blockKeys := []string{"b1", "b2", "b3", "b4", "b5"}

	hitmap := map[string][]string{
		"b1": {"pod-x"},
		"b2": {"pod-x"},
		"b3": {"pod-y"},
		"b4": {"pod-x"},
		"b5": {"pod-z"},
	}

	expected := map[string]int{
		"pod-x": 4,
		"pod-y": 3,
		"pod-z": 5,
	}

	scored, err := scorer.Score(blockKeys, hitmap)
	assert.NoError(t, err)
	for pod, score := range scored {
		assert.Equal(t, expected[pod], score)
	}
}

// TestCoverageBasedScorer verifies scoring based on total number of non-consecutive block hits.
func TestCoverageBasedScorer(t *testing.T) {
	scorer := &kvcache.CoverageBasedScorer{}
	blockKeys := []string{"b1", "b2", "b3", "b4", "b5"}

	hitmap := map[string][]string{
		"b1": {"pod-x"},
		"b2": {"pod-x"},
		"b3": {"pod-y"},
		"b4": {"pod-x"},
		"b5": {"pod-y"},
	}

	expected := map[string]int{
		"pod-x": 3,
		"pod-y": 2,
	}

	scored, err := scorer.Score(blockKeys, hitmap)
	assert.NoError(t, err)
	for pod, score := range scored {
		assert.Equal(t, expected[pod], score)
	}
}
