package llm_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/stretchr/testify/require"
)

func TestChunckOffsets(t *testing.T) {
	raw, err := os.ReadFile("./test_text001.txt")
	require.NoError(t, err, "Failed to read test text file")
	text := string(raw)
	require.NotEmpty(t, text, "Test text should not be empty")

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				offsets, err := llm.ChunckOffsets(text, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "ChunckOffsets should not return an error")
				builder := strings.Builder{}
				for _, off := range offsets {
					_, _, unique, _ := llm.ExtractChunk(text, off)
					builder.WriteString(unique)
				}
				require.Equal(t, text, builder.String(), "Rebuilt text should match original text")
			},
		)
	}
}

func TestChunckParagraphsOffsets(t *testing.T) {
	raw, err := os.ReadFile("./test_text002.txt")
	require.NoError(t, err, "Failed to read test text file")

	paragraphs := strings.Split(string(raw), "\n\n")
	require.NotEmpty(t, paragraphs, "Test paragraphs should not be empty")
	for i, p := range paragraphs {
		paragraphs[i] = strings.TrimSpace(p)
	}

	article := strings.Join(paragraphs, "")
	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				offsets, err := llm.ChunckParagraphsOffsets(paragraphs, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "ChunckParagraphsOffsets should not return an error")
				builder := strings.Builder{}
				for _, off := range offsets {
					_, _, unique, _ := llm.ExtractChunk(article, off)
					builder.WriteString(unique)
				}
				require.Equal(t, article, builder.String(), "Rebuilt text should match original paragraphs")
			},
		)
	}
}

func TestChuncking(t *testing.T) {
	raw, err := os.ReadFile("./test_text001.txt")
	require.NoError(t, err, "Failed to read test text file")
	text := string(raw)
	require.NotEmpty(t, text, "Test text should not be empty")

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 4096, Overlap: 512},
		{ChunkSize: 1024, Overlap: 128},
		{ChunkSize: 512, Overlap: 64},
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}

	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				chunks, err := llm.Chunck(text, tc.ChunkSize, tc.Overlap)
				require.NoError(t, err, "Chuncking should not return an error")
				rebuild := strings.Builder{}
				for _, chunk := range chunks {
					rebuild.WriteString(chunk[1])
				}
				require.Equal(t, text, rebuild.String(), "Rebuilt text should match original text")
			},
		)
	}

	// Test with invalid parameters
	_, err = llm.Chunck("", 0, 0)
	require.Error(t, err, "Chuncking with invalid parameters should return an error")

	_, err = llm.Chunck(text, 32, 32)
	require.Error(t, err, "Chuncking with invalid overlap should return an error")
}

func TestChunckingParagraphs(t *testing.T) {
	raw, err := os.ReadFile("./test_text002.txt")
	require.NoError(t, err, "Failed to read test text file")

	paragraphs := strings.Split(string(raw), "\n\n")
	require.NotEmpty(t, paragraphs, "Test paragraphs should not be empty")
	for i, p := range paragraphs {
		paragraphs[i] = strings.TrimSpace(p)
	}

	tcs := []struct {
		ChunkSize int
		Overlap   int
	}{
		{ChunkSize: 256, Overlap: 32},
		{ChunkSize: 128, Overlap: 16},
		{ChunkSize: 64, Overlap: 8},
		{ChunkSize: 32, Overlap: 4},
		{ChunkSize: 16, Overlap: 2},
	}
	for _, tc := range tcs {
		t.Run(
			fmt.Sprintf("ChunkSize=%d,Overlap=%d", tc.ChunkSize, tc.Overlap),
			func(t *testing.T) {
				chunks, err := llm.ChunckParagraphs(paragraphs, 64, 16)
				require.NoError(t, err, "Chuncking paragraphs should not return an error")
				builder := strings.Builder{}
				for _, chunk := range chunks {
					builder.WriteString(chunk[1])
				}
				require.Equal(t, strings.Join(paragraphs, ""), builder.String(), "Rebuilt text should match original paragraphs")
			},
		)
	}
}
