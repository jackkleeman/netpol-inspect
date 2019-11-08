package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	rootCmd = &cobra.Command{
		Use:   "netpol-inspect",
		Short: "Tool to understand the effects of network policies",
	}
	namespace string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to look in, if needed")
	rootCmd.AddCommand(describeCmd)
	rootCmd.AddCommand(applyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadKubeConfig() (*kubernetes.Clientset, error) {
	var config *rest.Config

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		config = cfg
	} else {
		var kubeconfigPath string

		if os.Getenv("KUBECONFIG") != "" {
			kubeconfigPath = os.Getenv("KUBECONFIG")
		} else {
			kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}

		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
		config = cfg
	}
	return kubernetes.NewForConfig(config)
}
