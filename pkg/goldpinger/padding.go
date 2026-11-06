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

package goldpinger

import (
	"crypto/rand"
	"encoding/base64"
)

// maxPingPadSize is the largest padding a /ping response will carry. It must
// match the maximum of the pad parameter in swagger.yml.
const maxPingPadSize = 65536

// pingPadding is the filler that padded /ping responses are sliced from. It is
// random so that nothing on the path can compress it away, and base64 encoded
// so that it is sent as-is inside the JSON response.
var pingPadding = func() string {
	raw := make([]byte, base64.RawStdEncoding.DecodedLen(maxPingPadSize)+1)
	rand.Read(raw)
	return base64.RawStdEncoding.EncodeToString(raw)[:maxPingPadSize]
}()

// GetPadding returns size bytes of filler for a padded /ping response, clamped
// to maxPingPadSize. It returns an empty string if size is not positive.
func GetPadding(size int32) string {
	if size <= 0 {
		return ""
	}
	return pingPadding[:min(int(size), maxPingPadSize)]
}
