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
	"encoding/json"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"kubesphere.io/telemetry/pkg/telemetry/collector"
	"kubesphere.io/telemetry/pkg/telemetry/report"
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
	config     *rest.Config
	collectors []collector.Collector
	report     report.Report
}

func (t *telemetry) RegisterCollector(cs ...collector.Collector) {
	t.collectors = append(t.collectors, cs...)
}

// Option is a configuration option supplied to NewTelemetry.
type Option func(*telemetry)

// WithClient set kubernetes client to collector data.
func WithConfig(config *rest.Config) Option {
	return func(t *telemetry) {
		t.config = config
	}
}

// WithReport set kubernetes client to collector data.
func WithReport(report report.Report) Option {
	return func(t *telemetry) {
		t.report = report
	}
}

func (t *telemetry) Start(ctx context.Context) error {
	cli, err := runtimeclient.New(t.config, runtimeclient.Options{
		Scheme: collector.Schema,
	})
	if err != nil {
		return err
	}
	var data = make(map[string]interface{})
	data["ts"] = time.Now().UTC().Format(time.RFC3339)
	//var wg wait.Group
	var wg errgroup.Group
	for _, c := range t.collectors {
		lc := c
		wg.Go(func() error {
			value, err := lc.Collect(ctx, cli)
			if err != nil {
				// retry
				klog.Errorf("collector %s collect data error %v", lc.RecordKey(), err)
				return err
			}
			data[lc.RecordKey()] = value
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return err
	}
	dataMap, err := serializeMap(data)
	if err != nil {
		klog.Errorf("failed to serializeMap %v", err)
	}
	return t.report.Save(ctx, dataMap)
}

func serializeMap(data map[string]any) (map[string]any, error) {
	bs, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string]any)
	return dataMap, json.Unmarshal(bs, &dataMap)
}
