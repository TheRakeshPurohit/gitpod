// Copyright (c) 2020 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	// yaml "gopkg.in/yaml.v2"

	// "io/ioutil"
	// "log"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/api/meta"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	// "io/fs"
	"io/ioutil"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	// "io/ioutil"
	"os"
	// "path/filepath"
	// "gopkg.in/yaml.v2"
	// "k8s.io/apimachinery/pkg/api/meta"
	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	// "k8s.io/client-go/kubernetes/scheme"
	// "k8s.io/client-go/kubernetes/scheme"
)

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "",
	Args:  cobra.ExactArgs(0),
	Run: func(_ *cobra.Command, args []string) {
		objs, err := parseAsJsonArray()
		if err != nil {
			log.Panic(err)
		}

		// sort by .kind and .metadata.name
		sort.SliceStable(objs, func(i, j int) bool {
			id := func(i int) string {
				return fmt.Sprintf("%s:%s", objs[i].GetKind(), objs[i].GetName())
			}
			return id(i) < id(j)
		})

		for _, obj := range objs {
			// filter out generic stuff: .status, .metadata.annotations, etc.
			filterGenericStuff(&obj)

			// handle specific objects
			// handle(&obj)
		}

		// pretty print to stdout
		bytes, err := json.MarshalIndent(objs, "", "  ")
		if err != nil {
			log.Panic(fmt.Errorf("unable to print output: %w", err))
		}
		fmt.Print(string(bytes))
	},
}

func parseAsJsonArray() (objs []unstructured.Unstructured, err error) {
	inBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %s", err)
	}

	err = json.Unmarshal(inBytes, &objs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return objs, nil
}

var emptyMap = map[string]string{}
func filterGenericStuff(obj *unstructured.Unstructured) {
	// no .metadata.annotations
	obj.SetAnnotations(emptyMap)
	obj.SetNamespace("")
    // # | yq d - '[*].metadata.creationTimestamp' \
	obj.SetCreationTimestamp(v1.Time{})

	obj.SetLabels(filterLabels(obj.GetLabels()))

	switch obj.GetKind() {
	case "Service":
		svc := asService(obj)
		svc.Spec.Selector = filterLabels(svc.Spec.Selector)
	case "Deployment":
		dep := asDeployment(obj)
		dep.Spec.Template.ObjectMeta.Labels = filterLabels(dep.Spec.Template.ObjectMeta.Labels)
		dep.Spec.Template.ObjectMeta.CreationTimestamp = v1.Time{}
		dep.Spec.Template.CreationTimestamp = v1.Time{}
		dep.Spec.Template.Spec.ImagePullSecrets = nil
	}

	// no .status
	delete(obj.Object, "status")
	delete(obj.Object, "automountServiceAccountToken")
	delete(obj.Object, "imagePullSecrets")
}

func filterLabels(lbls map[string]string) map[string]string {
	res := map[string]string{}
	for k, v := range lbls {
		switch {
		case k == "stage":
			continue
		case k == "kind":
			continue
		case k == "chart":
			continue
		case k == "heritage":
			continue
		case k == "release":
			continue
		case strings.HasPrefix(k, "helm.sh/"):
			continue
		case strings.HasPrefix(k, "app.kubernetes.io/"):
			continue
		}
		res[k] = v
	}
	return res
}

func asDeployment(obj *unstructured.Unstructured) *appsv1.Deployment {
	dep := appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &dep)
	assertNoError(err)
	return &dep
}

func asService(obj *unstructured.Unstructured) *corev1.Service {
	svc := corev1.Service{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &svc)
	assertNoError(err)
	return &svc
}

func handle(obj *unstructured.Unstructured) {
	id := fmt.Sprintf("%s:%s", obj.GetKind(), obj.GetName())

	//fmt.Printf("--%s:%s\n", strings.ToUpper(obj.GetKind()), obj.GetName())

	switch id {
	case "ConfigMap:content-service-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:auth-providers-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "auth-providers.json")
	case "ConfigMap:blobserve-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:image-builder-mk3-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "image-builder.json")
	case "ConfigMap:proxy-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		fmt.Printf("vhost.empty:\n%s", cm.Data["vhost.empty"])
		fmt.Printf("vhost.kedge:\n%s", cm.Data["vhost.kedge"])
		fmt.Printf("vhost.open-vsx:\n%s", cm.Data["vhost.open-vsx"])
		fmt.Printf("vhost.payment-endpoint:\n%s", cm.Data["vhost.payment-endpoint"])
	case "ConfigMap:registry-facade-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:restarter-scripts":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		fmt.Printf("run.sh:\n%s", cm.Data["run.sh"])
	case "ConfigMap:server-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:workspace-template":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data,
			"default.yaml",
			"imagebuild.yaml",
			"prebuild.yaml",
			"probe.yaml",
			"regular.yaml",
		)
	case "ConfigMap:ws-daemon-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:ws-manager-bridge-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "ws-manager-bridge.json")
	case "ConfigMap:ws-manager-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "ConfigMap:ws-proxy-config":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		printJsonDataFields(cm.Data, "config.json")
	case "Secret:db-sync-config":
		var sc *corev1.Secret
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sc)
		assertNoError(err)
		printJsonDataFields(convertToStringMap(sc.Data), "db-sync-gitpod.json", "db-sync-sessions.json")
	case "Secret:kedge-config":
		var sc *corev1.Secret
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sc)
		assertNoError(err)
		printJsonDataFields(convertToStringMap(sc.Data), "config.json")
	case "Secret:kedge-config-gitpod":
		var sc *corev1.Secret
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sc)
		assertNoError(err)
		printJsonDataFields(convertToStringMap(sc.Data), "config.json")
		// case *appsv1.DaemonSet:
		// 	fmt.Printf("object is a daemonset: %s\n", v.Name)
		// case *appsv1.StatefulSet:
		// 	fmt.Printf("object is a daemonset: %s\n", v.Name)
		// case *corev1.ConfigMap:
		// 	fmt.Printf("object is a configmap: %s\n", v.Name)
		// case *corev1.ServiceAccount:
		// 	fmt.Printf("object is a serviceaccount: %s\n", v.Name)
		// case *corev1.Secret:
		// 	fmt.Printf("object is a secret: %s\n", v.Name)
		// case *corev1.Service:
		// 	fmt.Printf("object is a service: %s\n", v.Name)
		// case *corev1.ResourceQuota:
		// 	fmt.Printf("object is a resourcequota: %s\n", v.Name)
		// case *batchv1.Job:
		// 	fmt.Printf("object is a job: %s\n", v.Name)
		// case *batchv1.CronJob:
		// 	fmt.Printf("object is a cronjob: %s\n", v.Name)
		// case *rbacv1.ClusterRole:
		// 	fmt.Printf("object is a clusterrole: %s\n", v.Name)
		// case *rbacv1.ClusterRoleBinding:
		// 	fmt.Printf("object is a clusterrolebinding: %s\n", v.Name)
		// case *rbacv1.Role:
		// 	fmt.Printf("object is a role: %s\n", v.Name)
		// case *rbacv1.RoleBinding:
		// 	fmt.Printf("object is a rolebinding: %s\n", v.Name)
		// case *policyv1beta1.PodSecurityPolicy:
		// 	fmt.Printf("object is a podsecuritypolicy: %s\n", v.Name)
		// case *policyv1.PodDisruptionBudget:
		// 	fmt.Printf("object is a poddisruptionbudget: %s\n", v.Name)
		// case *networkingv1.NetworkPolicy:
		// 	fmt.Printf("object is a networkpolicy: %s\n", v.Name)
		// default:
		// 	fmt.Printf("\nunhandled object kind: %s\n", id)
	}
}

func convertToStringMap(in map[string][]byte) map[string]string {
	m := map[string]string{}
	for k := range in {
		m[k] = string(in[k])
	}
	return m
}

func printJsonDataFields(m map[string]string, fields ...string) {
	for _, field := range fields {
		// fmt.Printf("--%s\n", field)
		var v interface{}
		err := json.Unmarshal([]byte(m[field]), &v)
		assertNoError(err)
		b, err := json.MarshalIndent(v, "", " ")
		assertNoError(err)
		fmt.Println(string(b))
	}
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.AddCommand(filterCmd)
}
