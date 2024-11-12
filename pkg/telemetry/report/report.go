/*
Copyright 2024 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package report

import (
	"context"
	"net/http"

	"golang.org/x/time/rate"
)

// save data to Report
type Report interface {
	Save(ctx context.Context, data map[string]any) error
}

// KSCloudClient rate limit http client to ksCloud
var KSCloudClient = newRateLimitedHTTPClient(5, 10)

func newRateLimitedHTTPClient(rps int, burst int) *http.Client {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return &http.Client{
		Transport: &rateLimitedTransport{
			Transport:   http.DefaultTransport,
			RateLimiter: limiter,
		},
	}
}

type rateLimitedTransport struct {
	Transport   http.RoundTripper
	RateLimiter *rate.Limiter
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.RateLimiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return t.Transport.RoundTrip(req)
}
