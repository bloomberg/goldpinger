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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

func doDNSProbe(addr string, timeout time.Duration) error {
	resolver := net.Resolver{}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ips, err := resolver.LookupHost(ctx, addr)
	if len(ips) == 0 {
		return fmt.Errorf("%s was resolved to 0 ips", addr)
	}
	return err
}

func doTCPProbe(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if conn != nil {
		defer conn.Close()
	}
	return err
}

func doHTTPProbe(addr string, timeout time.Duration) error {
	client := http.Client{Timeout: timeout}
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url scheme: '%s' in address", u.Scheme)
	}
	if u.Scheme == "https" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	resp, err := client.Get(addr)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s returned non-200 resp: %d", addr, resp.StatusCode)
	}
	return err
}
