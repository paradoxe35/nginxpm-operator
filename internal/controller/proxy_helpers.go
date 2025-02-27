package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cespare/xxhash"
	corev1 "k8s.io/api/core/v1"

	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GenerateNginxUpstreamName(rName, rNamespace string, servicePort int32, hostIPS []string) string {
	name := strings.Join([]string{rName, rNamespace}, "-")
	name = strings.TrimSuffix(name, "-")

	h := xxhash.New()
	h.Write([]byte(strings.Join(hostIPS, "-")))
	h.Write([]byte(fmt.Sprintf("%d", servicePort)))

	ipsHash := fmt.Sprintf("%x", h.Sum(nil))

	baseName := fmt.Sprintf("%s-%s", name, ipsHash)

	return fmt.Sprintf("%s-%s", nginxpm.NGINX_LB_SERVER_PREFIX, baseName)
}

func GetPodsHostIPS(ctx context.Context, r client.Reader, pods *corev1.PodList) []string {
	log := log.FromContext(ctx)

	var nodeIPs []string
	for _, pod := range pods.Items {
		// Skip pods that aren't running
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Get the node name the pod is running on
		nodeName := pod.Spec.NodeName

		// If you have direct access to the node object, you can get its IP:
		node := &corev1.Node{}
		if err := r.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
			log.Error(err, "Failed to get node", "nodeName", nodeName)
			continue
		}

		// Get the node's IP address (typically from InternalIP)
		var nodeIP string
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeIP = address.Address
				break
			}
		}

		if !slices.Contains(nodeIPs, nodeIP) {
			nodeIPs = append(nodeIPs, nodeIP)
		}
	}

	return nodeIPs
}

// Helper function to check if a pod matches a service's selector
func PodMatchesServiceSelector(pod *corev1.Pod, svc *corev1.Service) bool {
	// If the pod is being deleted, it's no longer part of the service
	if pod.DeletionTimestamp != nil {
		return false
	}

	// Get the service's selector
	selector := labels.SelectorFromSet(svc.Spec.Selector)

	// Check if the pod's labels match the selector
	return selector.Matches(labels.Set(pod.Labels))
}
