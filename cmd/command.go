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

package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"kubesphere.io/telemetry/pkg/telemetry"
	"kubesphere.io/telemetry/pkg/telemetry/report"
)

const (
	ENV_HISTORY_RETENTION   = "TELEMETRY_HISTORY_RETENTION"
	defaultHistoryRetention = 365 * 24 * time.Hour
)

type telemetryOptions struct {
	url     string
	cloudID string
	// clusterInfo live time. valid when product is kse.
	historyRetention time.Duration
}

func defaultTelemetryOptions() *telemetryOptions {
	// get history retension from env
	hr, err := time.ParseDuration(os.Getenv(ENV_HISTORY_RETENTION))
	if err != nil || hr == 0 {
		hr = defaultHistoryRetention
	}
	return &telemetryOptions{
		historyRetention: hr,
	}
}

func NewTelemetryCommand(version string) *cobra.Command {
	o := defaultTelemetryOptions()

	cmd := &cobra.Command{
		Use:     "telemetry",
		Long:    "telemetry cluster-info and send to cloud",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// get cli
			// set report
			var reporter report.Report
			if o.url == "" {
				reporter = report.NewLocalReport()
			} else { // sync to cloud
				rt, err := report.NewCloudReport(o.url, o.cloudID, o.historyRetention, config.GetConfigOrDie())
				if err != nil {
					return err
				}
				reporter = rt
			}
			return telemetry.NewTelemetry(telemetry.WithConfig(config.GetConfigOrDie()), telemetry.WithReport(reporter)).Start(signals.SetupSignalHandler())
		},
	}
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().StringVar(&o.url, "url", o.url, "the url for kubesphere cloud")
	cmd.Flags().StringVar(&o.cloudID, "cloud-id", o.cloudID, "the id for kubesphere cloud")
	cmd.Flags().DurationVar(&o.historyRetention, "history-retention", o.historyRetention, "how long the clusterInfo crd retention. ")
	cmd.AddCommand(versionCmd(version))
	return cmd
}

// Execute invokes the command.
func Execute(version string) error {
	if err := NewTelemetryCommand(version).Execute(); err != nil {
		return fmt.Errorf("error executing command: %+v", err)
	}
	return nil
}
