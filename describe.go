package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	typeNone = iota
	typeIngress
	typeEgress
	typeBoth
)

var describeCmd = &cobra.Command{
	Use:   "describe NAME",
	Short: "Prints information about the effect of a network policy",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := describeExisting(args[0]); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func describeExisting(name string) error {
	client, err := loadKubeConfig()
	if err != nil {
		return err
	}

	np, err := client.NetworkingV1().NetworkPolicies(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return describe(client, np)
}

func describe(client *kubernetes.Clientset, np *networkingv1.NetworkPolicy) error {
	ingress, egress := getPolicyType(np.Spec)

	selector, err := metav1.LabelSelectorAsSelector(&np.Spec.PodSelector)
	if err != nil {
		return err
	}

	applicablePods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return err
	}

	if len(applicablePods.Items) == 0 {
		fmt.Printf("%s does not apply to any running pods; has no effect\n", np.Name)
		return nil
	}

	// Get a list of all network policies that could possibly apply to these pods
	nps, err := client.NetworkingV1().NetworkPolicies(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	npsMap := map[string]networkingv1.NetworkPolicy{}

	for _, np := range nps.Items {
		npsMap[np.Name] = np
	}

	// Delete the one we are looking at
	for i, otherNp := range nps.Items {
		if otherNp.Name == np.Name {
			nps.Items = append(nps.Items[:i], nps.Items[i+1:]...)
			break
		}
	}
	before := podGroups(npsMap, applicablePods.Items, nps.Items)

	switch {
	case ingress && egress:
		// If we are ingress and egress, then we will isolate pods that were previously none, ingress, egress, but not both
		printIfNonZero("Would allow all ingress and egress if not for this whitelist", before[typeNone])
		printIfNonZero("Would allow all egress if not for this whitelist, may be allowed new ingress:", before[typeIngress])
		printIfNonZero("Would allow all ingress if not for this whitelist, may be allowed new egress:", before[typeEgress])
		printIfNonZero("May be allowed new ingress or egress:", before[typeBoth])
	case ingress:
		printIfNonZero("Would allow all ingress if not for this whitelist:", append(before[typeNone], before[typeEgress]...))
		printIfNonZero("May be allowed new ingress:", append(before[typeBoth], before[typeIngress]...))
	case egress:
		printIfNonZero("Would allow all egress if not for this whitelist:", append(before[typeNone], before[typeIngress]...))
		printIfNonZero("May be allowed new egress:", append(before[typeBoth], before[typeEgress]...))
	}

	return nil
}

func printIfNonZero(message string, list []string) {
	if len(list) == 0 {
		return
	}

	fmt.Println(message)
	fmt.Printf("  %s\n", list)
}

func podGroups(npsMap map[string]networkingv1.NetworkPolicy, pods []corev1.Pod, nps []networkingv1.NetworkPolicy) map[int][]string {
	selectors := map[string][]string{}

	for _, otherNp := range nps {
		selector, err := metav1.LabelSelectorAsSelector(&otherNp.Spec.PodSelector)
		if err != nil {
			panic(err)
		}
		sel := selector.String()

		selectors[sel] = append(selectors[sel], otherNp.Name)
	}

	podGroups := map[int][]string{}

	for _, pod := range pods {
		var ingress, egress bool

		// For every pod, try each selector.
		for selector, policies := range selectors {
			if !testLabelsAgainstSelector(pod.Labels, selector) {
				continue
			}

			for _, p := range policies {
				i, e := getPolicyType(npsMap[p].Spec)
				ingress = ingress || i
				egress = egress || e
			}
		}

		typ := typeNone
		switch {
		case ingress && egress:
			typ = typeBoth
		case ingress:
			typ = typeIngress
		case egress:
			typ = typeEgress
		}

		podGroups[typ] = append(podGroups[typ], pod.Name)
	}

	return podGroups

}

func testLabelsAgainstSelector(l map[string]string, selector string) bool {
	s, err := labels.Parse(selector)
	if err != nil {
		panic(err)
	}

	return s.Matches(labels.Set(l))
}

func getPolicyType(nps networkingv1.NetworkPolicySpec) (bool, bool) {
	if len(nps.PolicyTypes) == 0 {
		return true, len(nps.Egress) > 0
	}

	var ingress, egress bool
	for _, pt := range nps.PolicyTypes {
		switch pt {
		case networkingv1.PolicyTypeIngress:
			ingress = true
		case networkingv1.PolicyTypeEgress:
			egress = true
		}
	}

	return ingress, egress
}
