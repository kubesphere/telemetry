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

package telemetry

import (
	"context"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubesphere.io/telemetry/pkg/telemetry/collector"
	"kubesphere.io/telemetry/pkg/telemetry/report"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTelemetry(opts ...Option) manager.Runnable {
	t := &telemetry{
		collectors: collector.Registered,
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

type telemetry struct {
	client     runtimeclient.Client
	collectors []collector.Collector
	report     report.Report

	cloudID string
}

func (t *telemetry) RegisterCollector(cs ...collector.Collector) {
	t.collectors = append(t.collectors, cs...)
}

// Option is a configuration option supplied to NewTelemetry.
type Option func(*telemetry)

// WithClient set kubernetes client to collector data.
func WithClient(cli runtimeclient.Client) Option {
	return func(t *telemetry) {
		t.client = cli
	}
}

// WithReport set kubernetes client to collector data.
func WithReport(report report.Report) Option {
	return func(t *telemetry) {
		t.report = report
	}
}

func (t *telemetry) Start(ctx context.Context) error {
	var data = make(map[string]interface{})
	data["ts"] = time.Now().UTC().Format(time.RFC3339)
	var collectionOpt = &collector.CollectorOpts{Client: t.client, Ctx: ctx}
	var wg wait.Group
	for _, c := range t.collectors {
		lc := c
		wg.StartWithContext(ctx, func(ctx context.Context) {
			value, err := lc.Collect(collectionOpt)
			if err != nil {
				// retry
				klog.Errorf("collector %s collect data error %v", lc.RecordKey(), err)
				return
			}
			data[lc.RecordKey()] = value
		})
	}
	wg.Wait()

	return t.report.Save(ctx, data)
}
