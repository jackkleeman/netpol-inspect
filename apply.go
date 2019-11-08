package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
)

var applyFile string

func init() {
	applyCmd.PersistentFlags().StringVarP(&applyFile, "file", "f", "", "path to a network policy file")

}

var applyCmd = &cobra.Command{
	Use:   "apply -f file.yaml",
	Short: "Prints information about the effect of a network policy manifest",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := apply(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func apply() error {
	client, err := loadKubeConfig()
	if err != nil {
		return err
	}

	np, err := parseYAML(applyFile)
	if err != nil {
		return err
	}

	// Infer namespace from manifest, if provided
	if np.Namespace != "" {
		namespace = np.Namespace
	}

	return describe(client, np)
}

func parseYAML(filename string) (*networkingv1.NetworkPolicy, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", filename, err)
	}
	defer r.Close()

	decode := scheme.Codecs.UniversalDeserializer().Decode
	reader := bufio.NewReader(r)
	yamlReader := yaml.NewYAMLReader(reader)
	var k8sObject runtime.Object
	for {
		readBytes, err := yamlReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		k8sObject, _, err = decode(readBytes, nil, nil)
		if err != nil {
			return nil, err
		}

		switch object := k8sObject.(type) {
		case *networkingv1.NetworkPolicy:
			return object, nil
		}
	}

	return nil, fmt.Errorf("could not find a network policy in file %s", filename)
}
