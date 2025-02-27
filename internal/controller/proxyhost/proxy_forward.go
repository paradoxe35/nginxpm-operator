package proxyhost

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
	ProxyHost               *nginxpmoperatoriov1.ProxyHost
	UpstreamForward         *ProxyHostForward
	Forward                 nginxpmoperatoriov1.Forward
	UnscopedConfigSupported bool
	Label                   string
}

func (r *ProxyHostReconciler) makeForward(option MakeForwardOption) (*ProxyHostForward, error) {
	log := log.FromContext(option.Ctx)

	forward := option.Forward
	req := option.Req
	ctx := option.Ctx
	label := option.Label

	// Check if forward host or service is provided
	if forward.Host == nil && forward.Service == nil {
		err := fmt.Errorf("no forward host or service is provided, one of them is required, label: %s", label)
		log.Error(err, "no forward host or service is provided, one of them is required, label: %s", label)
		return nil, err
	}

	var proxyHostForward *ProxyHostForward

	// When forward service is provided
	if forward.Service != nil && forward.Host == nil {
		log.Info(fmt.Sprintf("Service resource is provided, finding service from Service resource, label: %s", label))

		service := &corev1.Service{}

		// If namespace is not provided, use the namespace of the proxyhost
		namespace := req.Namespace
		if forward.Service.Namespace != nil {
			namespace = *forward.Service.Namespace
		}
		// Retrieve the Service resource
		if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: forward.Service.Name}, service); err != nil {
			// If the Service resource is not found, we will not be able to create the forward
			log.Error(err, fmt.Sprintf("Service resource not found, please check the Service resource name, label: %s", label))
			return nil, err
		}

		// Extract service IP
		var serviceIP string
		servicePort := 0

		nginxUpstreamConfigs := make(map[string]string)

		// When the service type is NodePort
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			nodePortConfig, err := r.forwardWhenNodePortType(ctx, option.ProxyHost, service, forward)
			if err != nil {
				return nil, err
			}

			serviceIP = nodePortConfig.serviceIP
			servicePort = nodePortConfig.servicePort

			// We set can serviceIP to loadBalancer Name only when UnscopedConfigSupported is true
			// Means the Nginx Proxy Manager supports the UnscopedConfig
			if nodePortConfig.nginxUpstreamName != "" && option.UnscopedConfigSupported {
				nginxUpstreamConfigs[nodePortConfig.nginxUpstreamName] = nodePortConfig.nginxUpstreamConfig

				// Add also the nginx-upstream-config config to upstream forward exist
				if option.UpstreamForward != nil {
					if option.UpstreamForward.NginxUpstreamConfigs == nil {
						option.UpstreamForward.NginxUpstreamConfigs = make(map[string]string)
					}

					option.UpstreamForward.NginxUpstreamConfigs[nodePortConfig.nginxUpstreamName] = nodePortConfig.nginxUpstreamConfig
				}

				// Handle this only on root upstream forward (When UpstreamForward is nil)
				if option.UpstreamForward == nil {
					serviceIP = nodePortConfig.nginxUpstreamName
				}
			}

		} else {
			// When the service type is not NodePort
			serviceIP, servicePort = r.forwardWhenNotNodePortType(service, forward)
		}

		// Verify if service port is valid
		if servicePort == 0 {
			msg := fmt.Sprintf("service port is not valid, please check the Service resource name, label: %s", label)
			err := errors.New(msg)
			log.Error(err, msg)
			return nil, err
		}

		// Verify if service IP is valid
		if serviceIP == "" {
			msg := fmt.Sprintf("service IP is not valid, please check the Service resource name, label: %s", label)
			err := errors.New(msg)
			log.Error(err, msg)
			return nil, err
		}

		proxyHostForward = &ProxyHostForward{
			Scheme:               forward.Scheme,
			Host:                 serviceIP,
			Port:                 int(servicePort),
			AdvancedConfig:       forward.AdvancedConfig,
			NginxUpstreamConfigs: nginxUpstreamConfigs,
		}
	}

	// When forward host is provided
	if forward.Host != nil {
		log.Info(fmt.Sprintf("Host configuration is provided, applying to proxy host, label: %s", label))

		proxyHostForward = &ProxyHostForward{
			Scheme:               forward.Scheme,
			Host:                 forward.Host.HostName,
			Port:                 int(forward.Host.HostPort),
			AdvancedConfig:       forward.AdvancedConfig,
			NginxUpstreamConfigs: map[string]string{},
		}
	}

	if proxyHostForward == nil {
		return nil, fmt.Errorf("no forward host or service is provided, one of them is required, label: %s", label)
	}

	return proxyHostForward, nil
}

func (r *ProxyHostReconciler) forwardWhenNotNodePortType(service *corev1.Service, forward nginxpmoperatoriov1.Forward) (string, int) {
	if service.Spec.Type == corev1.ServiceTypeNodePort {
		return "", 0
	}

	serviceIP := service.Spec.ClusterIP
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			serviceIP = service.Status.LoadBalancer.Ingress[0].IP
		}
	}

	matchPort := func(ports []corev1.ServicePort, scheme string) int32 {
		scheme = strings.ToLower(scheme)
		for _, port := range ports {
			portName := strings.ToLower(port.Name)
			contains := strings.Contains(portName, scheme)

			if scheme == "http" {
				if contains && !strings.Contains(portName, "https") {
					return port.Port
				}
			} else if strings.Contains(portName, scheme) {
				return port.Port
			}
		}
		return 0
	}

	// Extract service port
	var servicePort int32
	if forward.Service.Port != nil {
		servicePort = *forward.Service.Port
	} else {
		servicePort = matchPort(service.Spec.Ports, "http")

		if forward.Scheme == "https" || servicePort == 0 {
			servicePort = matchPort(service.Spec.Ports, "https")
		}

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

func (r *ProxyHostReconciler) forwardWhenNodePortType(ctx context.Context, ph *nginxpmoperatoriov1.ProxyHost, service *corev1.Service, forward nginxpmoperatoriov1.Forward) (*nodePortConfig, error) {
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

	serviceIP := service.Spec.ClusterIP

	matchPort := func(ports []corev1.ServicePort, scheme string) int32 {
		scheme = strings.ToLower(scheme)
		for _, port := range ports {
			portName := strings.ToLower(port.Name)
			contains := strings.Contains(portName, scheme)

			if scheme == "http" {
				if contains && !strings.Contains(portName, "https") {
					return port.NodePort
				}
			} else if strings.Contains(portName, scheme) {
				return port.NodePort
			}
		}
		return 0
	}

	// Extract service port
	servicePort := matchPort(service.Spec.Ports, "http")
	if forward.Scheme == "https" || servicePort == 0 {
		servicePort = matchPort(service.Spec.Ports, "https")
	}
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
		ph.Name, ph.Namespace,
		servicePort, nodeIPs,
	)

	return &nodePortConfig{
		serviceIP:           serviceIP,
		servicePort:         int(servicePort),
		nginxUpstreamName:   conf.Name,
		nginxUpstreamConfig: conf.Config,
	}, nil
}

func mergeNginxUpstreamConfigs(configs map[string]string) string {
	var values []string
	for _, config := range configs {
		values = append(values, config)
	}

	return strings.Join(values, "\n")
}
