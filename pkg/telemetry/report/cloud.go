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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

const defaultTelemetryEndpoint = "/apis/telemetry/v1/clusterinfos?cluster_id=${cluster_id}"

func NewCloudReport(cloudURL string, cloudID string) Report {
	return &cloudReport{cloudURL: cloudURL, cloudID: cloudID}
}

type cloudReport struct {
	cloudURL string
	cloudID  string
}

func (r cloudReport) Save(ctx context.Context, data map[string]interface{}) error {
	klog.Infof("Send data to kubesphere cloud")
	data["cloudId"] = r.cloudID
	// convert req data
	reqData, err := json.Marshal(data)
	if err != nil {
		klog.Errorf("convert clusterInfo data status to json error %v", err)
		return err
	}
	clusterId := ""
	for _, cluster := range data["clusters"].([]map[string]any) {
		if cluster["role"] == "host" {
			clusterId = cluster["nid"].(string)
		}
	}
	if clusterId == "" { // When the data has not been collected yet
		klog.Infof("clusterId is empty. skip sync")
		return nil
	}

	telemetryReq := fmt.Sprintf(`{ "user_id": "%s","data": %s }`, r.cloudID, string(reqData))
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", r.cloudURL, strings.ReplaceAll(defaultTelemetryEndpoint, "${cluster_id}", clusterId)), bytes.NewBufferString(telemetryReq))
	if err != nil {
		klog.Errorf("new request for cloud error %v", err)
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		klog.Errorf("do request for cloud error %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp code expect %v, but get code %v ", http.StatusOK, resp.StatusCode)
	}
	return nil
}
