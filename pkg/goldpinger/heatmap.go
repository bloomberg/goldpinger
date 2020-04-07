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
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"sort"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func addLabel(img *image.RGBA, x, y int, text string) {
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{25, 200, 25, 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)},
	}
	drawer.DrawString(text)
}

// Calculates the color of the box to draw based on the latency and tresholds
// We are aiming at slightly more palatable colors than just moving from 255 green to 255 red,
// so we will use 25B, and then move from (25R, 200G) to (200R, 25G), so our scale is effectively 350 points
func getPingBoxColor(latency int64, tresholdLatencies [3]int64) *color.RGBA {
	var red, green uint8 = 25, 200
	if latency > tresholdLatencies[2] {
		red, green = 200, 25
	} else if latency >= tresholdLatencies[1] {
		red, green = 200, 200
		diff := (float32(latency-tresholdLatencies[1]) / float32(tresholdLatencies[2]-tresholdLatencies[1])) * 175
		green = green - uint8(diff)
	} else if latency >= tresholdLatencies[0] {
		red, green = 25, 200
		diff := (float32(latency-tresholdLatencies[0]) / float32(tresholdLatencies[1]-tresholdLatencies[0])) * 175
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

	// parse the query to set the parameters
	query := r.URL.Query()

	ctx, cancel := context.WithTimeout(
		r.Context(),
		time.Duration(GoldpingerConfig.CheckAllTimeoutMs)*time.Millisecond,
	)
	defer cancel()

	// get the results
	checkResults := CheckAllPods(ctx, GetAllPods())

	// set some sizes
	numberOfPods := len(checkResults.Responses)
	legendSize := 200
	boxSize := 14
	paddingSize := 1
	heatmapSize := numberOfPods*(boxSize+paddingSize) + boxSize*2
	tresholdLatencies := [3]int64{1, 10, 100}
	for index := range tresholdLatencies {
		stringValue := query["t"+fmt.Sprintf("%d", index)]
		if len(stringValue) == 0 {
			continue
		}
		if v, err := strconv.ParseInt(stringValue[0], 0, 64); err == nil && v >= 0 {
			tresholdLatencies[index] = v
		}
	}

	canvas := image.NewRGBA(image.Rect(0, 0, heatmapSize+legendSize, heatmapSize))

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
		if *results.OK {
			for destinationIP, response := range results.Response.PodResults {
				x, y := getPingBoxCoordinates(order[sourceIP], order[destinationIP], boxSize, paddingSize)
				color := getPingBoxColor(response.ResponseTimeMs, tresholdLatencies)
				drawPingBox(canvas, boxSize+x, boxSize+y, boxSize, color)
			}
		}
	}

	// draw the legend
	for index, ip := range keys {
		// ip
		addLabel(canvas, heatmapSize, (index+1)*(boxSize+paddingSize)+13, fmt.Sprintf("%d", index)+": "+ip)
		// rows
		addLabel(canvas, 0, (index+1)*(boxSize+paddingSize)+13, fmt.Sprintf("%d", index))
		// columns
		addLabel(canvas, (index+1)*(boxSize+paddingSize), 13, fmt.Sprintf("%d", index))
	}

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, canvas); err != nil {
		zap.L().Error("error encoding png", zap.Error(err))
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		zap.L().Error("error writing heatmap buffer out", zap.Error(err))
	}
}
