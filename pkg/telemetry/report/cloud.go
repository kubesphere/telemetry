package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	CRDGroupVersionKind     = schema.GroupVersionKind{Group: "telemetry.kubesphere.io", Version: "v1alpha1", Kind: "ClusterInfo"}
	CRDListGroupVersionKind = schema.GroupVersionKind{Group: "telemetry.kubesphere.io", Version: "v1alpha1", Kind: "ClusterInfoList"}
)

const (
	CRDKind     = "ClusterInfo"
	CRDResource = "clusterinfoes"

	ProductKSE = "kse"
	ProductKS  = "ks"

	defaultTelemetryEndpoint = "/apis/telemetry/v1/clusterinfos?cluster_id=${cluster_id}"
)

func NewCloudReport(cloudURL string, cloudID string, historyRetention time.Duration, config *restclient.Config) (Report, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	client, err := runtimeclient.New(config, runtimeclient.Options{})
	if err != nil {
		return nil, err
	}
	return &cloudReport{
		cloudURL:         cloudURL,
		cloudID:          cloudID,
		historyRetention: historyRetention,
		client:           client,
		discoveryClient:  discoveryClient,
	}, nil
}

type cloudReport struct {
	cloudURL         string
	cloudID          string
	historyRetention time.Duration
	client           runtimeclient.Client
	discoveryClient  discovery.DiscoveryInterface
}

// Save implements Report. save to crd(ClusterInfo). and report history crd to cloud.
func (k cloudReport) Save(ctx context.Context, data map[string]any) error {
	// check env
	apiresources, err := k.discoveryClient.ServerPreferredResources()
	if err != nil {
		return err
	}
	for _, apiresource := range apiresources {
		if apiresource.GroupVersion == CRDGroupVersionKind.GroupVersion().String() {
			return k.saveWithCRD(ctx, data)
		}
	}

	data["product"] = ProductKS
	return k.syncToCloud(ctx, data)
}

func (k cloudReport) saveWithCRD(ctx context.Context, data map[string]any) error {
	// save current data to a new crd
	if err := k.saveCRD(ctx, data); err != nil {
		return err
	}
	// delete expired crd
	if err := k.expiredCRD(ctx); err != nil {
		return err
	}
	// sync crd to cloud
	return k.syncCRD(ctx)
}

func (k *cloudReport) saveCRD(ctx context.Context, data map[string]any) error {
	clusterInfo := &unstructured.Unstructured{}
	clusterInfo.SetGroupVersionKind(CRDGroupVersionKind)
	ts, err := time.Parse(time.RFC3339, data["ts"].(string))
	if err != nil {
		return err
	}
	clusterInfo.SetName(ts.UTC().Format("20060102150405"))
	// create crd
	if err := k.client.Create(ctx, clusterInfo); err != nil {
		return err
	}
	// set status to crd
	// update clusterInfo status
	newClusterInfo := clusterInfo.DeepCopy()
	if err := unstructured.SetNestedMap(newClusterInfo.Object, data, "status"); err != nil {
		return err
	}
	return k.client.Status().Patch(ctx, newClusterInfo, runtimeclient.MergeFrom(clusterInfo.DeepCopy()))
}

func (k *cloudReport) expiredCRD(ctx context.Context) error {
	clusterInfoList := &unstructured.UnstructuredList{}
	clusterInfoList.SetGroupVersionKind(CRDListGroupVersionKind)
	if err := k.client.List(ctx, clusterInfoList); err != nil {
		return err
	}
	var err error
	for _, clusterInfo := range clusterInfoList.Items {
		if clusterInfo.GetCreationTimestamp().Add(k.historyRetention).Before(time.Now()) {
			err = errors.Join(err, k.client.Delete(ctx, &clusterInfo))
		}
	}
	return err
}

func (k *cloudReport) syncCRD(ctx context.Context) error {
	clusterInfoList := &unstructured.UnstructuredList{}
	clusterInfoList.SetGroupVersionKind(CRDListGroupVersionKind)
	if err := k.client.List(ctx, clusterInfoList); err != nil {
		return err
	}
	var errs error
	for _, clusterInfo := range clusterInfoList.Items {
		if clusterInfo.GetDeletionTimestamp() != nil { // ctd is deleted
			continue
		}
		if _, found, err := unstructured.NestedFieldCopy(clusterInfo.Object, "status", "syncTime"); err == nil && found { // crd is synced
			continue
		}

		data, found, err := unstructured.NestedMap(clusterInfo.Object, "status")
		if err != nil || !found {
			errs = errors.Join(errs, fmt.Errorf("failed to get status from %s. error is %v or not found", clusterInfo.GetName(), err))
		}
		data["product"] = ProductKSE
		err = k.syncToCloud(ctx, data)
		if err != nil { // sync failed
			errs = errors.Join(errs, fmt.Errorf("failed to sync %s to cloud. error is %v", clusterInfo.GetName(), err))
		} else { // sync success. add syncTime to clusterInfo
			newClusterInfo := clusterInfo.DeepCopy()
			if err := unstructured.SetNestedField(newClusterInfo.Object, metav1.Now().UTC().Format(time.RFC3339), "status", "syncTime"); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to set syncTime filed in %s. error is %v", clusterInfo.GetName(), err))
			}
			if err := k.client.Status().Patch(ctx, newClusterInfo, runtimeclient.MergeFrom(clusterInfo.DeepCopy())); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to patch syncTime filed in %s. error is %v", clusterInfo.GetName(), err))
			}
		}
	}
	return errs
}

func (k *cloudReport) syncToCloud(ctx context.Context, data map[string]any) error {
	// get clusterId from data
	clusterId := ""
	for _, cluster := range data["clusters"].([]any) {
		if cluster.(map[string]any)["role"] == "host" {
			clusterId = cluster.(map[string]any)["nid"].(string)
		}
	}
	if clusterId == "" { // When the data has not been collected yet
		klog.Infof("clusterId is empty. skip sync")
		return nil
	}
	data["cloudId"] = k.cloudID

	// convert req data
	reqData, err := json.Marshal(data)
	if err != nil {
		klog.Errorf("convert clusterInfo data status to json error %v", err)
		return err
	}

	telemetryReq := fmt.Sprintf(`{ "user_id": "%s","data": %s }`, k.cloudID, string(reqData))
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", k.cloudURL, strings.ReplaceAll(defaultTelemetryEndpoint, "${cluster_id}", clusterId)), bytes.NewBufferString(telemetryReq))
	if err != nil {
		klog.Errorf("new request for cloud error %v", err)
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := KSCloudClient.Do(request)
	if err != nil {
		klog.Errorf("do request for cloud error %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp code expect %v, but get code %v ", http.StatusOK, resp.StatusCode)
	}
	klog.Infof("Send data to kubesphere cloud success")
	return nil
}
