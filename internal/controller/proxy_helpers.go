package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cespare/xxhash"
	corev1 "k8s.io/api/core/v1"

	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NginxUpstreamHost struct {
	Hostname string // IP or Hostname
	Port     int32
}

func generateNginxUpstreamName(rName, rNamespace string, hosts []NginxUpstreamHost) string {
	name := strings.Join([]string{rName, rNamespace}, "-")
	name = strings.TrimSuffix(name, "-")

	strHosts := ""
	for _, host := range hosts {
		strHosts += fmt.Sprintf("%s-%d", host.Hostname, host.Port)
	}

	h := xxhash.New()
	h.Write([]byte(strHosts))

	ipsHash := fmt.Sprintf("%x", h.Sum(nil))

	baseName := fmt.Sprintf("%s-%s", name, ipsHash)

	return fmt.Sprintf("%s-%s", nginxpm.NGINX_LB_SERVER_PREFIX, baseName)
}

type upstreamConfig struct {
	Name   string
	Config string
}

const (
	keepaliveCount    = 32               // Reasonable pool size per worker
	keepaliveTimeout  = 60 * time.Second // Nginx default is 60s
	keepaliveRequests = 1000             // Nginx default is 1000 since 1.19.10 (prev 100)
	maxFails          = 3                // Times to fail before marking down
	failTimeout       = 30 * time.Second // Duration to mark down
)

func GenerateNginxUpstreamConfig(rName, rNamespace string, hosts []NginxUpstreamHost) upstreamConfig {
	nginxUpstreamName := ""
	nginxUpstreamConfig := ""

	if len(hosts) > 0 {
		nginxUpstreamName = generateNginxUpstreamName(rName, rNamespace, hosts)

		nginxUpstreamConfig = fmt.Sprintf("upstream %s {\n", nginxUpstreamName)
		nginxUpstreamConfig += "    least_conn;\n"
		// keepalive config
		nginxUpstreamConfig += fmt.Sprintf("    keepalive %d;\n", keepaliveCount)
		nginxUpstreamConfig += fmt.Sprintf("    keepalive_timeout %ds;\n", int(keepaliveTimeout.Seconds()))
		nginxUpstreamConfig += fmt.Sprintf("    keepalive_requests %d;\n", keepaliveRequests)
		nginxUpstreamConfig += "\n" // Blank line for readability

		failTimeoutStr := fmt.Sprintf("%ds", int(failTimeout.Seconds()))
		for _, host := range hosts {
			nginxUpstreamConfig += fmt.Sprintf("    server %s:%d max_fails=%d fail_timeout=%s;\n",
				host.Hostname,
				host.Port,
				maxFails,
				failTimeoutStr,
			)
		}
		nginxUpstreamConfig += "}"
	}

	return upstreamConfig{
		Name:   nginxUpstreamName,
		Config: nginxUpstreamConfig,
	}
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
