/*
Copyright 2023 The KubeSphere Authors.

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

package collector

import (
	"context"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var Registered []Collector

// Collector the telemetry data
type Collector interface {
	// RecordKey  telemetry data key
	RecordKey() string
	// Collect telemetry data value
	Collect(*CollectorOpts) (interface{}, error)
}

type CollectorOpts struct {
	Client runtimeClient.Client
	Ctx    context.Context
}

func register(collector Collector) {
	Registered = append(Registered, collector)
}
