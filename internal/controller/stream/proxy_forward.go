package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/internal/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MakeForwardOption struct {
	Ctx                     context.Context
	Req                     ctrl.Request
	Stream                  *nginxpmoperatoriov1.Stream
	UnscopedConfigSupported bool
}

func (r *StreamReconciler) makeForward(option MakeForwardOption) (*StreamForward, error) {
	log := log.FromContext(option.Ctx)

	req := option.Req
	ctx := option.Ctx
	forward := option.Stream.Spec.Forward

	// Check if forward host or service is provided
	if forward.Host == nil && forward.Service == nil {
		err := fmt.Errorf("no forward host or service is provided, one of them is required")
		log.Error(err, "no forward host or service is provided, one of them is required")
		return nil, err
	}

	var streamForward *StreamForward

	// When forward service is provided
	if forward.Service != nil && forward.Host == nil {
		log.Info("Service resource is provided, finding service from Service resource")

		service := &corev1.Service{}

		// If namespace is not provided, use the namespace of the stream
		namespace := req.Namespace
		if forward.Service.Namespace != nil {
			namespace = *forward.Service.Namespace
		}
		// Retrieve the Service resource
		if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: forward.Service.Name}, service); err != nil {
			// If the Service resource is not found, we will not be able to create the forward
			log.Error(err, "Service resource not found, please check the Service resource name")
			return nil, err
		}

		// Extract service IP
		var serviceIP string
		servicePort := 0

		var nginxUpstreamConfigs string

		// When the service type is NodePort
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			nodePortConfig, err := r.forwardWhenNodePortType(ctx, option.Stream, service)
			if err != nil {
				return nil, err
			}

			serviceIP = nodePortConfig.serviceIP
			servicePort = nodePortConfig.servicePort

			// We can set serviceIP to loadBalancer Name only when UnscopedConfigSupported is true
			// Means the Nginx Proxy Manager supports the UnscopedConfig
			if nodePortConfig.nginxUpstreamName != "" && option.UnscopedConfigSupported {
				nginxUpstreamConfigs = nodePortConfig.nginxUpstreamConfig
				serviceIP = nodePortConfig.nginxUpstreamName
			}

		} else {
			// When the service type is not NodePort
			serviceIP, servicePort = r.forwardWhenNotNodePortType(service, forward)
		}

		// Verify if service port is valid
		if servicePort == 0 {
			msg := "service port is not valid, please check the Service resource name"
			err := errors.New(msg)
			log.Error(err, msg)
			return nil, err
		}

		// Verify if service IP is valid
		if serviceIP == "" {
			msg := "service IP is not valid, please check the Service resource name"
			err := errors.New(msg)
			log.Error(err, msg)
			return nil, err
		}

		streamForward = &StreamForward{
			Host:                 serviceIP,
			Port:                 int(servicePort),
			NginxUpstreamConfigs: nginxUpstreamConfigs,
		}
	}

	// When forward host is provided
	if forward.Host != nil {
		log.Info("Host configuration is provided, applying to stream")

		streamForward = &StreamForward{
			Host: forward.Host.HostName,
			Port: int(forward.Host.HostPort),
		}
	}

	if streamForward == nil {
		return nil, fmt.Errorf("no forward host or service is provided, one of them is required")
	}

	return streamForward, nil
}

func (r *StreamReconciler) forwardWhenNotNodePortType(service *corev1.Service, forward nginxpmoperatoriov1.StreamForward) (string, int) {
	if service.Spec.Type == corev1.ServiceTypeNodePort {
		return "", 0
	}

	serviceIP, servicePort := getServiceDestination(service, forward, func(ports []corev1.ServicePort, scheme string) int32 {
		for _, port := range ports {
			if portMatched(scheme, port) {
				return port.Port
			}
		}
		return 0
	})

	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			serviceIP = service.Status.LoadBalancer.Ingress[0].IP
		}
	}

	if forward.Service.Port != nil {
		servicePort = *forward.Service.Port
	} else {
		if servicePort == 0 && len(service.Spec.Ports) > 0 {
			servicePort = service.Spec.Ports[0].Port
		}
	}

	return serviceIP, int(servicePort)
}

type nodePortConfig struct {
	serviceIP           string
	servicePort         int
	nginxUpstreamName   string
	nginxUpstreamConfig string
}

func (r *StreamReconciler) forwardWhenNodePortType(ctx context.Context, st *nginxpmoperatoriov1.Stream, service *corev1.Service) (*nodePortConfig, error) {
	forward := st.Spec.Forward

	if service.Spec.Type != corev1.ServiceTypeNodePort {
		return nil, fmt.Errorf("service type is not NodePort")
	}

	// Get pods using the service selector
	pods := &corev1.PodList{}
	listOps := &client.ListOptions{
		Namespace:     service.Namespace,
		LabelSelector: labels.SelectorFromSet(service.Spec.Selector),
	}

	if err := r.List(ctx, pods, listOps); err != nil {
		return nil, err
	}

	serviceIP, servicePort := getServiceDestination(service, forward, func(ports []corev1.ServicePort, scheme string) int32 {
		for _, port := range ports {
			if portMatched(scheme, port) {
				return port.NodePort
			}
		}
		return 0
	})

	if servicePort == 0 && len(service.Spec.Ports) > 0 {
		servicePort = service.Spec.Ports[0].NodePort
	}

	// Get the host IPs of the pods
	nodeIPs := controller.GetPodsHostIPS(ctx, r, pods)

	// Save the first node IP as the service IP
	if len(nodeIPs) > 0 {
		serviceIP = nodeIPs[0]
	}

	conf := controller.GenerateNginxUpstreamConfig(
		st.Name, st.Namespace,
		servicePort, nodeIPs,
	)

	return &nodePortConfig{
		serviceIP:           serviceIP,
		servicePort:         int(servicePort),
		nginxUpstreamName:   conf.Name,
		nginxUpstreamConfig: conf.Config,
	}, nil
}

type MatchPort func(ports []corev1.ServicePort, scheme string) int32

func portMatched(scheme string, port corev1.ServicePort) bool {
	scheme = strings.ToLower(scheme)
	protocol := strings.ToLower(string(port.Protocol))
	portName := strings.ToLower(port.Name)

	return scheme == protocol || strings.Contains(portName, scheme)
}

func getServiceDestination(service *corev1.Service, forward nginxpmoperatoriov1.StreamForward, matchPort MatchPort) (string, int32) {
	serviceIP := service.Spec.ClusterIP

	var servicePort int32
	if forward.TCPForwarding {
		servicePort = matchPort(service.Spec.Ports, "TCP")
	}

	if forward.UDPForwarding {
		if v := matchPort(service.Spec.Ports, "UDP"); v != 0 {
			servicePort = v
		}
	}

	return serviceIP, servicePort
}
