// Copyright (c) 2020 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"encoding/json"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	// "io/ioutil"
	// "log"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// filterCmd represents the inject command
var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		filter(args[0])
	},
}

func filter(dir string) {
	deserJSON()
	deserYAML()
}

func deserJSON() {

	input := `
	{
		"apiVersion": "v1",
		"kind": "ConfigMap",
		"metadata": {
           "name": "Test"
    	},
		"data": {
			"config.json": "{ \"name\": \"lalalala\" }"
	  	}
	}
	`

	// read single JSON object
	obj := unstructured.Unstructured{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		panic(err)
	}
	fmt.Printf("name: %s", obj.GetName())

	// TODO gather all items in a list
	// TODO to sorting based on GetKind/GetName
	// TODO do filtering of labels on the generic unstructured objects (GetLabels/SetLabels)

	// Then decent into special-handling per specific object
	handle(obj)
}

func deserYAML() {

	input := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: Test
data:
  config.json: test
`

	obj := unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(input), &obj)
	if err != nil {
		panic(err)
	}

	fmt.Printf("name: %s", obj.GetName())
}

func handle(obj unstructured.Unstructured) {

	id := fmt.Sprintf("%s:%s", obj.GetKind(), obj.GetName())

	switch id {
	case "ConfigMap:Test":
		var cm *corev1.ConfigMap
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(obj.Object, &cm)
		assertNoError(err)
		fmt.Printf("content of config.json: %s", cm.Data["config.json"])
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
	default:
		fmt.Printf("unhandled object kind: %s\n", id)
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
