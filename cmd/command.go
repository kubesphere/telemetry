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
	"fmt"

	"github.com/spf13/cobra"
	"kubesphere.io/telemetry/pkg/telemetry/collector"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"kubesphere.io/telemetry/pkg/telemetry"
	"kubesphere.io/telemetry/pkg/telemetry/report"
)

type telemetryOptions struct {
	url     string
	cloudID string
}

func defaultTelemetryOptions() *telemetryOptions {
	return &telemetryOptions{}
}

func NewTelemetryCommand(version string) *cobra.Command {
	o := defaultTelemetryOptions()

	cmd := &cobra.Command{
		Use:     "telemetry",
		Long:    "telemetry cluster-info and send to cloud",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// get cli
			cli, err := runtimeclient.New(config.GetConfigOrDie(), runtimeclient.Options{
				Scheme: collector.Schema,
			})
			if err != nil {
				return err
			}

			rt := report.NewLocalReport()
			if o.url != "" {
				rt = report.NewCloudReport(o.url, o.cloudID)
			}
			return telemetry.NewTelemetry(telemetry.WithClient(cli), telemetry.WithReport(rt)).Start(signals.SetupSignalHandler())
		},
	}
	cmd.Flags().StringVar(&o.url, "url", o.url, "the url for kubesphere cloud")
	cmd.Flags().StringVar(&o.cloudID, "cloud-id", o.cloudID, "the id for kubesphere cloud")

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
