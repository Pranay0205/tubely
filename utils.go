package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
)

const horizontalRatio float64 = 1.77777777778
const vertialRatio float64 = 0.5625

func containsMediaType(s string, substring []string) bool {
	for _, sub := range substring {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var buffer bytes.Buffer

	cmd.Stdout = &buffer

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("coudn't run the ffprobe command: %w", err)
	}

	var videoInf VideoInfo
	err = json.Unmarshal(buffer.Bytes(), &videoInf)

	if err != nil {
		return "", fmt.Errorf("coudn't unmarshal the stream: %w", err)
	}

	if len(videoInf.Streams) == 0 {
		return "", fmt.Errorf("stream not present: %w", err)
	}

	width := videoInf.Streams[0].Width
	height := videoInf.Streams[0].Height
	ratio := float64(width) / float64(height)

	tolerance := 0.1
	if math.Abs(ratio-horizontalRatio) < tolerance {
		return "16:9", nil
	}

	if math.Abs(ratio-vertialRatio) < tolerance {
		return "9:16", nil
	}

	return "other", nil
}
