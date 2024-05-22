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
	"fmt"
	"time"

	corev1alpha1 "kubesphere.io/api/core/v1alpha1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	register(&Extension{})
}

type Extension struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Ctime   string `json:"ctime"`
}

func (e Extension) RecordKey() string {
	return "extension"
}

func (e Extension) Collect(ctx context.Context, client runtimeClient.Client) (interface{}, error) {
	subsList := &corev1alpha1.InstallPlanList{}
	err := client.List(ctx, subsList)
	if err != nil {
		return nil, fmt.Errorf("get SubscriptionList error %v", err)
	}
	// statistic extension data
	resData := make([]Extension, len(subsList.Items))
	for i, s := range subsList.Items {
		resData[i] = Extension{
			Name:    s.Spec.Extension.Name,
			Version: s.Spec.Extension.Version,
			Ctime:   s.CreationTimestamp.Local().Format(time.RFC3339),
		}
	}
	return resData, nil
}
