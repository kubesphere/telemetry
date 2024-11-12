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

	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	corev1alpha1 "kubesphere.io/api/core/v1alpha1"
	iamv1beta1 "kubesphere.io/api/iam/v1beta1"
	tenantv1beta1 "kubesphere.io/api/tenant/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var Registered []Collector

// Collector the telemetry data
type Collector interface {
	// RecordKey  telemetry data key
	RecordKey() string
	// Collect telemetry data value
	Collect(ctx context.Context, client runtimeclient.Client) (interface{}, error)
}

func register(collector Collector) {
	Registered = append(Registered, collector)
}

var Schema = runtime.NewScheme()

func init() {
	// register scheme
	runtimeutil.Must(clusterv1alpha1.AddToScheme(Schema))
	runtimeutil.Must(corev1alpha1.AddToScheme(Schema))
	runtimeutil.Must(tenantv1beta1.AddToScheme(Schema))
	runtimeutil.Must(iamv1beta1.AddToScheme(Schema))
}
