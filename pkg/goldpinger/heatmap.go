// Copyright 2018 Bloomberg Finance L.P.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is safe to edit. Once it exists it will not be overwritten

package goldpinger

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"sort"
	"strconv"
)

// Calculates the color of the box to draw based on the latency and tresholds
// We are aiming at slightly more palatable colors than just moving from 255 green to 255 red,
// so we will use 25B, and then move from (25R, 200G) to (200R, 25G), so our scale is effectively 350 points
func getPingBoxColor(latency, tresholdA, tresholdB, tresholdC int64) *color.RGBA {
	var red, green uint8 = 25, 200
	if latency > tresholdC {
		red, green = 200, 25
	} else if latency > tresholdB {
		red, green = 200, 200
		diff := (float32(latency-tresholdB) / float32(tresholdC-tresholdB)) * 175
		green = green - uint8(diff)
	} else if latency > tresholdA {
		red, green = 25, 200
		diff := (float32(latency-tresholdA) / float32(tresholdB-tresholdA)) * 175
		red = red + uint8(diff)
	}
	return &color.RGBA{red, green, 25, 255}
}

func drawPingBox(img *image.RGBA, _x, _y, size int, color *color.RGBA) {
	for x := _x; x < _x+size; x++ {
		for y := _y; y < _y+size; y++ {
			img.Set(x, y, *color)
		}
	}
}

func getPingBoxCoordinates(col, row, boxSize, padding int) (int, int) {
	return col * (boxSize + padding), row * (boxSize + padding)
}

// HeatmapHandler returns a PNG with a heatmap representation
func HeatmapHandler(w http.ResponseWriter, r *http.Request) {

	// get the results
	checkResults := CheckAllPods(GetAllPods())

	// set some sizes
	numberOfPods := len(checkResults.Responses)
	boxSize := 20
	paddingSize := 1
	heatmapSize := numberOfPods * (boxSize + paddingSize)
	var tresholdLatencyA int64 = 50
	var tresholdLatencyB int64 = 100
	var tresholdLatencyC int64 = 200

	canvas := image.NewRGBA(image.Rect(0, 0, heatmapSize, heatmapSize))

	// establish an order and fix the max delay
	var keys []string
	for sourceIP := range checkResults.Responses {
		keys = append(keys, sourceIP)
	}
	sort.Strings(keys)
	order := make(map[string]int)
	for index, key := range keys {
		order[key] = index
	}

	// draw all the boxes
	for sourceIP, results := range checkResults.Responses {
		fmt.Println("source", sourceIP)
		fmt.Println("OK ?", *results.OK)

		keys = append(keys, sourceIP)

		if *results.OK {
			for destinationIP, response := range results.Response {
				x, y := getPingBoxCoordinates(order[sourceIP], order[destinationIP], boxSize, paddingSize)
				color := getPingBoxColor(response.ResponseTimeMs, tresholdLatencyA, tresholdLatencyB, tresholdLatencyC)
				drawPingBox(canvas, x, y, boxSize, color)
			}
		}
	}

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, canvas); err != nil {
		log.Println("error encoding png", err)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("error writing heatmap buffer out", err)
	}
}
