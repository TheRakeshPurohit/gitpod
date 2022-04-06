// Copyright (c) 2020 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"fmt"
	"io/fs"
	// "io/ioutil"
	// "log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	// "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	// "k8s.io/apimachinery/pkg/api/meta"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
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
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		fmt.Printf("reading file: %s\n", path)
		yaml, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read file: %s\n", path)
			return nil
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(yaml, nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to deserialize: %s\n", err)
			return nil
		}

		handle(obj)
		return nil
	})
}

func handle(obj runtime.Object) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		fmt.Printf("object is a deployment: %s\n", v.Name)
	case *appsv1.DaemonSet:
		fmt.Printf("object is a daemonset: %s\n", v.Name)
	case *appsv1.StatefulSet:
		fmt.Printf("object is a daemonset: %s\n", v.Name)
	case *corev1.ConfigMap:
		fmt.Printf("object is a configmap: %s\n", v.Name)
	case *corev1.ServiceAccount:
		fmt.Printf("object is a serviceaccount: %s\n", v.Name)
	case *corev1.Secret:
		fmt.Printf("object is a secret: %s\n", v.Name)
	case *corev1.Service:
		fmt.Printf("object is a service: %s\n", v.Name)
	case *corev1.ResourceQuota:
		fmt.Printf("object is a resourcequota: %s\n", v.Name)
	case *batchv1.Job:
		fmt.Printf("object is a job: %s\n", v.Name)
	case *batchv1.CronJob:
		fmt.Printf("object is a cronjob: %s\n", v.Name)
	case *rbacv1.ClusterRole:
		fmt.Printf("object is a clusterrole: %s\n", v.Name)
	case *rbacv1.ClusterRoleBinding:
		fmt.Printf("object is a clusterrolebinding: %s\n", v.Name)
	case *rbacv1.Role:
		fmt.Printf("object is a role: %s\n", v.Name)
	case *rbacv1.RoleBinding:
		fmt.Printf("object is a rolebinding: %s\n", v.Name)
	case *policyv1beta1.PodSecurityPolicy:
		fmt.Printf("object is a podsecuritypolicy: %s\n", v.Name)
	case *policyv1.PodDisruptionBudget:
		fmt.Printf("object is a poddisruptionbudget: %s\n", v.Name)
	case *networkingv1.NetworkPolicy:
		fmt.Printf("object is a networkpolicy: %s\n", v.Name)
	default:
		fmt.Printf("unhandled object kind: %s\n", v.GetObjectKind().GroupVersionKind().Kind)
	}
}

func init() {
	rootCmd.AddCommand(filterCmd)
}
