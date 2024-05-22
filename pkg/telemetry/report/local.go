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
	"encoding/json"
	"os"
	"time"

	"k8s.io/klog/v2"
)

func NewLocalReport() Report {
	return &localReport{}
}

type localReport struct {
}

func (r localReport) Save(ctx context.Context, data map[string]any) error {
	klog.Infof("Save data to local file in current dir")
	file, err := os.Create("clusterInfo-" + time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	reqData, err := json.Marshal(data)
	if err != nil {
		klog.Errorf("convert clusterInfo data status to json error %v", err)
		return err
	}

	_, err = file.Write(reqData)
	if err != nil {
		return err
	}

	return nil
}
