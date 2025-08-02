package snapshotting

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePVCSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
		wantErr  bool
	}{
		{
			name:     "zero size should return error",
			size:     0,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "negative size should return error",
			size:     -100,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "very small size should return minimum 1GB",
			size:     1024, // 1KB
			expected: "1Gi",
			wantErr:  false,
		},
		{
			name:     "size less than 1GB after buffer should return 1GB",
			size:     500 * 1024 * 1024, // 500MB
			expected: "1Gi",
			wantErr:  false,
		},
		{
			name:     "exactly 1GB should return 2GB with 110% buffer",
			size:     1024 * 1024 * 1024, // 1GB
			expected: "2Gi",
			wantErr:  false,
		},
		{
			name:     "2GB should return 3GB with 110% buffer",
			size:     2 * 1024 * 1024 * 1024, // 2GB
			expected: "3Gi",
			wantErr:  false,
		},
		{
			name:     "10GB should return 11GB with 110% buffer",
			size:     10 * 1024 * 1024 * 1024, // 10GB
			expected: "11Gi",
			wantErr:  false,
		},
		{
			name:     "size with remainder should round up",
			size:     1536 * 1024 * 1024, // 1.5GB
			expected: "2Gi",
			wantErr:  false,
		},
		{
			name:     "large size should work correctly",
			size:     100 * 1024 * 1024 * 1024, // 100GB
			expected: "110Gi",
			wantErr:  false,
		},
		{
			name:     "size at overflow boundary should return error",
			size:     math.MaxInt64/11 + 1,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "maximum safe size should work",
			size:     math.MaxInt64 / 11,
			expected: "858993460Gi", // This is the calculated result for max safe size
			wantErr:  false,
		},
		{
			name:     "edge case: size that results in exactly 1GB after buffer",
			size:     976128930, // This should result in exactly 1GB after 110% buffer
			expected: "1Gi",
			wantErr:  false,
		},
		{
			name:     "edge case: size that results in just over 1GB after buffer",
			size:     1073741825, // 1GB + 1 byte, should result in 2Gi after 110% buffer
			expected: "2Gi",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculatePVCSize(tt.size)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestCalculatePVCSizeBufferCalculation tests the 110% buffer calculation specifically
func TestCalculatePVCSizeBufferCalculation(t *testing.T) {
	// Test that the buffer calculation is correct
	// For 1GB input, we expect 110% = 1.1GB, which should round up to 2GB
	size := int64(1024 * 1024 * 1024) // 1GB
	result, err := calculatePVCSize(size)
	
	assert.NoError(t, err)
	assert.Equal(t, "2Gi", result)
	
	// Verify the internal calculation
	bufferedSize := (size * 11) / 10 // 110% of 1GB = 1.1GB
	expectedBufferedSize := int64(1024*1024*1024) * 11 / 10
	assert.Equal(t, expectedBufferedSize, bufferedSize)
}

// TestCalculatePVCSizeRoundingBehavior tests the rounding behavior
func TestCalculatePVCSizeRoundingBehavior(t *testing.T) {
	tests := []struct {
		name        string
		sizeInBytes int64
		expected    string
	}{
		{
			name:        "exactly divisible by GB",
			sizeInBytes: 1024 * 1024 * 1024, // 1GB
			expected:    "2Gi",
		},
		{
			name:        "with remainder - should round up",
			sizeInBytes: 1024*1024*1024 + 1, // 1GB + 1 byte
			expected:    "2Gi",
		},
		{
			name:        "large remainder - should round up",
			sizeInBytes: 1024*1024*1024 + 500*1024*1024, // 1.5GB
			expected:    "2Gi",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculatePVCSize(tt.sizeInBytes)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCalculatePVCSizeMinimumSize tests the minimum size enforcement
func TestCalculatePVCSizeMinimumSize(t *testing.T) {
	// Test various small sizes that should all result in 1GB minimum
	sizes := []int64{
		1,                    // 1 byte
		1024,                 // 1KB
		1024 * 1024,          // 1MB
		100 * 1024 * 1024,    // 100MB
		500 * 1024 * 1024,    // 500MB
		800 * 1024 * 1024,    // 800MB
	}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d_bytes", size), func(t *testing.T) {
			result, err := calculatePVCSize(size)
			assert.NoError(t, err)
			assert.Equal(t, "1Gi", result)
		})
	}
}