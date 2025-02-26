/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
)

const (
	proxyHostFinalizer = "proxyhost.nginxpm-operator.io/finalizers"

	PH_TOKEN_FIELD = ".spec.token.name"

	PH_CUSTOM_CERTIFICATE_FIELD = ".spec.ssl.customCertificate.name"

	PH_LETSENCRYPT_CERTIFICATE_FIELD = ".spec.ssl.letsEncryptCertificate.name"

	PH_FORWARD_SERVICE_FIELD = ".spec.forward.service.name"

	PH_CUSTOM_LOCATION_FORWARD_FIELD = ".spec.customLocations.forward.service.name"

	DEFAULT_EMAIL = "support@nginxpm-operator.io"
)

// ProxyHostReconciler reconciles a ProxyHost object
type ProxyHostReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type ProxyHostForward struct {
	Scheme               string
	Host                 string
	Port                 int
	AdvancedConfig       string
	NginxUpstreamConfigs map[string]string
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ProxyHost object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ProxyHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	ph := &nginxpmoperatoriov1.ProxyHost{}

	// Fetch the ProxyHost instance
	err := r.Get(ctx, req.NamespacedName, ph)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("proxyHost resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get proxyHost")
		return ctrl.Result{}, err
	}

	isMarkedToBeDeleted := !ph.ObjectMeta.DeletionTimestamp.IsZero()

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	if !isMarkedToBeDeleted {
		if err := AddFinalizer(r, ctx, proxyHostFinalizer, ph); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Let's just set the status as Unknown when no status is available
	if len(ph.Status.Conditions) == 0 {
		UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
			meta.SetStatusCondition(&ph.Status.Conditions, metav1.Condition{
				Status:  metav1.ConditionUnknown,
				Type:    "Reconciling",
				Reason:  "Reconciling",
				Message: "Starting reconciliation",
			})
		})
	}

	// Create a new Nginx Proxy Manager client
	nginxpmClient, err := InitNginxPMClient(ctx, r, req, ph.Spec.Token)
	if err != nil {
		// Stop reconciliation if the resource is marked for deletion and the client can't be created
		if isMarkedToBeDeleted {
			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, proxyHostFinalizer, ph); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		r.Recorder.Event(
			ph, "Warning", "InitNginxPMClient",
			fmt.Sprintf("Failed to init nginxpm client, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
		)

		// Set the status as False when the client can't be created
		UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
			meta.SetStatusCondition(&ph.Status.Conditions, metav1.Condition{
				Status:  metav1.ConditionFalse,
				Type:    "InitNginxPMClient",
				Reason:  "InitNginxPMClient",
				Message: err.Error(),
			})
		})

		return ctrl.Result{}, err
	}

	// Delete the ProxyHost record in the Nginx Proxy Manager instance before deleting the resource
	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(ph, proxyHostFinalizer) {
			log.Info("Performing Finalizer Operations for ProxyHost")

			// Delete the ProxyHost record in the Nginx Proxy Manager instance
			// If the ProxyHost is bound, we will not delete the record
			if ph.Status.Id != nil && !ph.Status.Bound {
				log.Info("Deleting ProxyHost record in Nginx Proxy Manager instance")
				err := nginxpmClient.DeleteProxyHost(int(*ph.Status.Id))

				if err != nil {
					log.Error(err, "Failed to delete ProxyHost record in Nginx Proxy Manager instance")
				}
			}

			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, proxyHostFinalizer, ph); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Domains should be unique
	_, err = r.domainsShouldBeUnique(ctx, ph)
	if err != nil {
		// Set the status as False when the client can't be created
		UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
			meta.SetStatusCondition(&ph.Status.Conditions, metav1.Condition{
				Status:  metav1.ConditionFalse,
				Type:    "DomainsShouldBeUnique",
				Reason:  "DomainsShouldBeUnique",
				Message: err.Error(),
			})
		})
		return ctrl.Result{}, err
	}

	// Create or update proxy host
	err = r.createOrUpdateProxyHost(ctx, req, ph, nginxpmClient)
	if err != nil {
		// Set the status as False when the client can't be created
		UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
			meta.SetStatusCondition(&ph.Status.Conditions, metav1.Condition{
				Status:  metav1.ConditionFalse,
				Type:    "CreateOrUpdateProxyHost",
				Reason:  "CreateOrUpdateProxyHost",
				Message: err.Error(),
			})
		})

		return ctrl.Result{}, err
	}

	// Set the status as True when the client can be created
	UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
		meta.SetStatusCondition(&ph.Status.Conditions, metav1.Condition{
			Status:  metav1.ConditionTrue,
			Type:    "CreateOrUpdateProxyHost",
			Reason:  "CreateOrUpdateProxyHost",
			Message: fmt.Sprintf("Proxy host created or updated, ResourceName: %s", req.Name),
		})
	})

	return ctrl.Result{}, nil
}

func (r *ProxyHostReconciler) domainsShouldBeUnique(ctx context.Context, ph *nginxpmoperatoriov1.ProxyHost) (bool, error) {
	log := log.FromContext(ctx)

	proxyHosts := &nginxpmoperatoriov1.ProxyHostList{}

	err := r.List(ctx, proxyHosts)
	if err != nil {
		log.Error(err, "Failed to list proxy hosts")
		return false, err
	}

	if len(proxyHosts.Items) == 0 {
		log.Info("No proxy hosts found, assuming domains should be unique")
		return true, nil
	}

	domains := r.toDomainList(ph)

	// add proxy hosts domains to the list
	for _, proxyHost := range proxyHosts.Items {
		if proxyHost.GetName() == ph.GetName() && proxyHost.GetNamespace() == ph.GetNamespace() {
			continue
		}

		proxyHostDomains := r.toDomainList(&proxyHost)
		// check if the domain is already used by another proxy host
		for _, domain := range domains {
			if slices.Contains(proxyHostDomains, domain) {
				msg := fmt.Sprintf("Domain %s is already used by another proxy host: (name: %s, namespace: %s)", domain, proxyHost.GetName(), proxyHost.GetNamespace())

				err := errors.New(msg)
				log.Error(err, msg)
				return false, err
			}
		}
	}

	return true, nil
}

func (r *ProxyHostReconciler) createOrUpdateProxyHost(ctx context.Context, req ctrl.Request, ph *nginxpmoperatoriov1.ProxyHost, nginxpmClient *nginxpm.Client) error {
	log := log.FromContext(ctx)

	var proxyHost *nginxpm.ProxyHost
	var err error

	bound := ph.Status.Bound

	// Convert domain names to []string
	domains := r.toDomainList(ph)

	// Let's check if the proxy host is already created
	if ph.Status.Id != nil {
		proxyHost, err = nginxpmClient.FindProxyHostByID(*ph.Status.Id)
		if err != nil {
			r.Recorder.Event(
				ph, "Warning", "FindProxyHostByID",
				fmt.Sprintf("Failed to find proxy host by ID, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)

			log.Error(err, "Failed to find proxy host by ID")
			return err
		}

	} else {
		// If finding by ID doesn't match a record, we search for the proxy host by domain.
		proxyHost, _ = nginxpmClient.FindProxyHostByDomain(domains)

		if proxyHost != nil {
			bound = true
		}
	}

	// ProxyHost forward operation
	proxyHostForward, err := r.makeForward(MakeForwardOption{
		Ctx:             ctx,
		Req:             req,
		ProxyHost:       ph,
		Forward:         ph.Spec.Forward,
		UpstreamForward: nil,
		Label:           "upstream-forward",
	})

	if err != nil {
		r.Recorder.Event(
			ph, "Warning", "MakeForward",
			fmt.Sprintf("Failed to make forward, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
		)
		return err
	}

	// Certificate operation
	var certificateID *int
	if ph.Spec.Ssl != nil {
		certificate, err := r.makeCertificate(ctx, req, ph, nginxpmClient)
		if err != nil {
			r.Recorder.Event(
				ph, "Warning", "MakeCertificate",
				fmt.Sprintf("Failed to make certificate, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)
			return err
		}

		if certificate != nil {
			certificateID = &certificate.ID
		}
	}

	// CustomLocation operation
	// pass the proxyHostForward to constructCustomLocation,
	// so that custom locations forward can pass their nginx-upstream-config to the upstream forward
	customLocations, err := r.constructCustomLocation(ctx, req, ph, proxyHostForward)
	if err != nil {
		r.Recorder.Event(
			ph, "Warning", "ConstructCustomLocation",
			fmt.Sprintf("Failed to construct custom locations, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
		)
		return err
	}

	input := nginxpm.ProxyHostRequestInput{
		DomainNames:           domains,
		ForwardHost:           proxyHostForward.Host,
		ForwardScheme:         proxyHostForward.Scheme,
		ForwardPort:           proxyHostForward.Port,
		AdvancedConfig:        proxyHostForward.AdvancedConfig,
		BlockExploits:         ph.Spec.BlockExploits,
		AllowWebsocketUpgrade: ph.Spec.WebsocketSupport,
		CachingEnabled:        ph.Spec.CachingEnabled,
		Locations:             customLocations,
		CustomFields:          make(nginxpm.ProxyHostRequestCustomFields),
	}

	// Handle custom fields
	withCustomFields := func(proxyHost *nginxpm.ProxyHost, input *nginxpm.ProxyHostRequestInput) bool {
		// Handle Unscoped custom field
		unscopedConfigSupported := nginxpm.JsonFieldInProxyHost(proxyHost, nginxpm.CUSTOM_FIELD_UNSCOPED_CONFIG)
		nginxUpstreamConfig := mergeNginxUpstreamConfigs(proxyHostForward.NginxUpstreamConfigs)

		if nginxUpstreamConfig != "" {
			// We are doing this for compatibility reasons
			input.CustomFields[nginxpm.CUSTOM_FIELD_UNSCOPED_CONFIG] = nginxpm.RequestCustomField{
				Field:   nginxpm.CUSTOM_FIELD_UNSCOPED_CONFIG,
				Value:   nginxUpstreamConfig,
				Allowed: unscopedConfigSupported,
			}
		}

		// All fields should be supported
		allFieldsSupported := true
		for _, custom := range input.CustomFields {
			if !custom.Allowed {
				allFieldsSupported = false
				break
			}
		}

		return allFieldsSupported
	}

	allCustomFieldsSupported := withCustomFields(proxyHost, &input)

	// Handle SSL fields
	if ph.Spec.Ssl != nil {
		input.SSLForced = ph.Spec.Ssl.SslForced
		input.HTTP2Support = ph.Spec.Ssl.Http2Support
		input.HSTSEnabled = ph.Spec.Ssl.HstsEnabled
		input.HSTSSubdomains = ph.Spec.Ssl.HstsSubdomains
	}

	if certificateID != nil {
		certId := int(*certificateID)
		input.CertificateID = &certId

		log.Info("Certificate ID found, applied to proxy host")
	} else {
		log.Info("No certificate found, ignoring certificate operation")
		input.CertificateID = nil
	}

	// Update proxy host
	if proxyHost != nil {
		proxyHost, err = nginxpmClient.UpdateProxyHost(proxyHost.ID, input)
		if err != nil {
			r.Recorder.Event(
				ph, "Warning", "UpdateProxyHost",
				fmt.Sprintf("Failed to update proxy host, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)

			log.Error(err, "Failed to update proxy host")
			return err
		}

		log.Info("ProxyHost updated successfully")
	} else {
		// Create proxy host
		proxyHost, err = nginxpmClient.CreateProxyHost(input)
		if err != nil {
			r.Recorder.Event(
				ph, "Normal", "CreateProxyHost",
				fmt.Sprintf("Failed to create proxy host, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)

			log.Error(err, "Failed to create proxy host")
			return err
		}

		// In case not all custom field supported, we send update request
		if !allCustomFieldsSupported {
			withCustomFields(proxyHost, &input) // call withCustomFields again to ensure all custom fields are supported
			nginxpmClient.UpdateProxyHost(proxyHost.ID, input)
		}

		log.Info("ProxyHost created successfully")
	}

	return UpdateStatus(ctx, r.Client, ph, req.NamespacedName, func() {
		ph.Status.Id = &proxyHost.ID
		ph.Status.CertificateId = certificateID
		ph.Status.Bound = bound
	})
}

// ############################################# CUSTOM LOCATION OPERATION ######################################

func (r *ProxyHostReconciler) constructCustomLocation(ctx context.Context, req ctrl.Request, ph *nginxpmoperatoriov1.ProxyHost, upstreamForward *ProxyHostForward) ([]nginxpm.ProxyHostLocation, error) {
	log := log.FromContext(ctx)

	customLocations := make([]nginxpm.ProxyHostLocation, len(ph.Spec.CustomLocations))

	for i, location := range ph.Spec.CustomLocations {
		forward, err := r.makeForward(MakeForwardOption{
			Ctx:             ctx,
			Req:             req,
			ProxyHost:       ph,
			UpstreamForward: upstreamForward,
			Forward:         location.Forward,
			Label:           fmt.Sprintf("downstream-forward-%d", i),
		})

		if err != nil {
			return nil, err
		}

		customLocations[i] = nginxpm.ProxyHostLocation{
			Path:           location.LocationPath,
			AdvancedConfig: location.Forward.AdvancedConfig,
			ForwardScheme:  forward.Scheme,
			ForwardHost:    forward.Host + location.Forward.Path,
			ForwardPort:    forward.Port,
		}
	}

	log.Info("CustomLocations found, applying to proxy host")

	return customLocations, nil
}

// ############################################# FORWARD OPERATION ##############################################

// ############################################# CERTIFICATE OPERATION ##############################################

func (r *ProxyHostReconciler) makeCertificate(ctx context.Context, req ctrl.Request, ph *nginxpmoperatoriov1.ProxyHost, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	if ph.Spec.Ssl == nil {
		log.Info("SSL is not enabled, skipping certificate operation")
		return nil, nil
	}

	var certificate *nginxpm.Certificate
	var err error

	// if LetsEncryptCertificate is provided, then find the certificate from Let's Encrypt resource
	if ph.Spec.Ssl.LetsEncryptCertificate != nil {
		log.Info("LetsEncryptCertificate is provided, finding certificate from LetsEncryptCertificate resource")
		certificate, err = r.getLetsEncryptCertificateFromReference(ctx, req, ph.Spec.Ssl.LetsEncryptCertificate, nginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	// if CustomCertificate is provided, then find the certificate from CustomCertificate resource
	if ph.Spec.Ssl.CustomCertificate != nil {
		log.Info("CustomCertificate is provided, finding certificate from CustomCertificate resource")
		certificate, err = r.getCustomCertificateFromReference(ctx, req, ph.Spec.Ssl.CustomCertificate, nginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	// if CertificateId is provided, then find the certificate from the ID
	if ph.Spec.Ssl.CertificateId != nil {
		log.Info("CertificateId is provided, finding certificate from ID")
		certificate, err = r.getCertificateFromID(ctx, *ph.Spec.Ssl.CertificateId, nginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	// If None of the above is provided and AutoCertificateRequest is enabled,
	// then we find or create a new certificate from Let's Encrypt
	if ph.Spec.Ssl.AutoCertificateRequest && certificate == nil {
		log.Info("Since no LetsEncryptCertificate, CustomCertificate, or CertificateId is provided, AutoCertificateRequest is enabled, finding or creating certificate")
		certificate, err = r.findOrCreateCertificate(ctx, ph, nginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	return certificate, nil
}

// Get certificate from Let's Encrypt reference
func (r *ProxyHostReconciler) getLetsEncryptCertificateFromReference(ctx context.Context, req ctrl.Request, reference *nginxpmoperatoriov1.SslLetsEncryptCertificate, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	lec := nginxpmoperatoriov1.LetsEncryptCertificate{}

	// If namespace is not provided, use the namespace of the proxyhost
	namespace := req.Namespace
	if reference.Namespace != nil {
		namespace = *reference.Namespace
	}

	// Retrieve the LetsEncryptCertificate resource
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reference.Name}, &lec); err != nil {
		// If the LetsEncryptCertificate resource is not found, we will not be able to create the certificate
		log.Error(err, "LetsEncryptCertificate resource not found, please check the LetsEncryptCertificate resource name")
		return nil, err
	}

	if lec.Status.Id == nil {
		log.Info("no certificate ID is provided, please check the LetsEncryptCertificate resource")
		return nil, fmt.Errorf("no certificate ID is provided, please check the LetsEncryptCertificate resource")
	}

	certificate, err := nginxpmClient.FindCertificateByID(*lec.Status.Id)
	if err != nil {
		log.Error(err, "Failed to find certificate by ID")
		return nil, err
	}

	return certificate, nil
}

// Get certificate from CustomCertificate reference
func (r *ProxyHostReconciler) getCustomCertificateFromReference(ctx context.Context, req ctrl.Request, reference *nginxpmoperatoriov1.SslCustomCertificate, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	customCert := nginxpmoperatoriov1.CustomCertificate{}

	// If namespace is not provided, use the namespace of the proxyhost
	namespace := req.Namespace
	if reference.Namespace != nil {
		namespace = *reference.Namespace
	}

	// Retrieve the CustomCertificate resource
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reference.Name}, &customCert); err != nil {
		// If the CustomCertificate resource is not found, we will not be able to create the certificate
		log.Error(err, "CustomCertificate resource not found, please check the CustomCertificate resource name")
		return nil, err
	}

	if customCert.Status.Id == nil {
		log.Info("no certificate ID is provided, please check the CustomCertificate resource")
		return nil, fmt.Errorf("no certificate ID is provided, please check the CustomCertificate resource")
	}

	certificate, err := nginxpmClient.FindCertificateByID(*customCert.Status.Id)
	if err != nil {
		log.Error(err, "Failed to find certificate by ID")
		return nil, err
	}

	return certificate, nil
}

// Get certificate from ID
func (r *ProxyHostReconciler) getCertificateFromID(ctx context.Context, id int, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	certificate, err := nginxpmClient.FindCertificateByID(id)
	if err != nil {
		log.Error(err, "Failed to find certificate by ID")
		return nil, err
	}

	return certificate, nil
}

// Find certificate by domain name
// If certificate is not found, create a new one from Let's Encrypt
func (r *ProxyHostReconciler) findOrCreateCertificate(ctx context.Context, ph *nginxpmoperatoriov1.ProxyHost, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	if ph.Spec.Ssl == nil {
		log.Info("SSL is not enabled, skipping certificate creation")
		return nil, nil
	}

	domains := r.toDomainList(ph)
	ssl := ph.Spec.Ssl

	letsEncryptEmail := DEFAULT_EMAIL
	if ssl.LetsEncryptEmail != nil && *ssl.LetsEncryptEmail != "" {
		letsEncryptEmail = *ssl.LetsEncryptEmail
	}

	certificate, err := nginxpmClient.FindCertificateByDomain(domains)
	if err != nil {
		log.Error(err, "[autoCertificateRequest] Failed to find certificate by domain")
		return nil, err
	}

	if certificate != nil {
		log.Info("[autoCertificateRequest] Certificate found, applying to proxy host")
	}

	// If certificate is not found, we will create a new one
	if certificate == nil {
		log.Info("[autoCertificateRequest] Certificate not found, creating new certificate...")
		lecCertificate, err := nginxpmClient.CreateLetEncryptCertificate(nginxpm.CreateLetEncryptCertificateRequest{
			DomainNames: domains,
			Meta: nginxpm.CreateLetEncryptCertificateRequestMeta{
				DNSChallenge:     false,
				LetsEncryptAgree: true,
				LetsEncryptEmail: letsEncryptEmail,
			},
		})
		if err != nil {
			log.Error(err, "[autoCertificateRequest] Failed to create certificate")
			return nil, err
		}

		certificate = &nginxpm.Certificate{
			ID:          lecCertificate.ID,
			CreatedOn:   lecCertificate.CreatedOn,
			ModifiedOn:  lecCertificate.ModifiedOn,
			Provider:    lecCertificate.Provider,
			NiceName:    lecCertificate.NiceName,
			DomainNames: lecCertificate.DomainNames,
			ExpiresOn:   lecCertificate.ExpiresOn,
		}
	}

	return certificate, nil
}

// ############################################# UTILS ##############################################

func (r *ProxyHostReconciler) toDomainList(ph *nginxpmoperatoriov1.ProxyHost) []string {
	domains := make([]string, len(ph.Spec.DomainNames))
	for i, domain := range ph.Spec.DomainNames {
		domains[i] = string(domain)
	}

	return domains
}

// ############################################# CONTROLLER ##############################################

// SetupWithManager sets up the controller with the Manager.
func (r *ProxyHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_TOKEN_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)

			if ph.Spec.Token == nil {
				// If token is not provided, use the default token name
				return []string{TOKEN_DEFAULT_NAME}
			}

			if ph.Spec.Token.Name == "" {
				return nil
			}

			return []string{ph.Spec.Token.Name}
		}); err != nil {
		return err
	}

	// Add the CustomCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_CUSTOM_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Ssl == nil || ph.Spec.Ssl.CustomCertificate == nil || ph.Spec.Ssl.CustomCertificate.Name == "" {
				return nil
			}
			return []string{ph.Spec.Ssl.CustomCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the LetsEncryptCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_LETSENCRYPT_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Ssl == nil || ph.Spec.Ssl.LetsEncryptCertificate == nil || ph.Spec.Ssl.LetsEncryptCertificate.Name == "" {
				return nil
			}
			return []string{ph.Spec.Ssl.LetsEncryptCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the Forward Service to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_FORWARD_SERVICE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Forward.Service == nil || ph.Spec.Forward.Service.Name == "" {
				return nil
			}
			return []string{ph.Spec.Forward.Service.Name}
		}); err != nil {
		return err
	}

	// Add the CustomLocation Forward Service to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_CUSTOM_LOCATION_FORWARD_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if len(ph.Spec.CustomLocations) == 0 {
				return nil
			}

			var fieldsList []string

			for _, location := range ph.Spec.CustomLocations {
				if location.Forward.Service == nil || location.Forward.Service.Name == "" {
					continue
				}

				// Append if not already present
				if !slices.Contains(fieldsList, location.Forward.Service.Name) {
					fieldsList = append(fieldsList, location.Forward.Service.Name)
				}
			}

			if len(fieldsList) == 0 {
				return nil
			}

			return fieldsList
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.ProxyHost{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Owns(&nginxpmoperatoriov1.CustomCertificate{}).
		Owns(&nginxpmoperatoriov1.LetsEncryptCertificate{}).
		Owns(&corev1.Service{}).
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(PH_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.CustomCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(PH_CUSTOM_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.LetsEncryptCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(PH_LETSENCRYPT_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(PH_FORWARD_SERVICE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(PH_CUSTOM_LOCATION_FORWARD_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *ProxyHostReconciler) findObjectsForMap(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.ProxyHostList{}

		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, object.GetName()),
		}

		err := r.List(ctx, attachedObjects, listOps)
		if err != nil {
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, len(attachedObjects.Items))
		for i, item := range attachedObjects.Items {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
		}

		return requests
	}
}
