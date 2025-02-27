package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RetrieveCertificateOption struct {
	Cxt           context.Context
	Req           ctrl.Request
	Reader        client.Reader
	NginxpmClient *nginxpm.Client

	LetsEncryptCertificate *nginxpmoperatoriov1.SslLetsEncryptCertificate
	CustomCertificate      *nginxpmoperatoriov1.SslCustomCertificate
	CertificateId          *int
}

func RetrieveCertificate(o RetrieveCertificateOption) (*nginxpm.Certificate, error) {
	log := log.FromContext(o.Cxt)

	var certificate *nginxpm.Certificate
	var err error

	// if LetsEncryptCertificate is provided, then find the certificate from Let's Encrypt resource
	if o.LetsEncryptCertificate != nil {
		log.Info("LetsEncryptCertificate is provided, finding certificate from LetsEncryptCertificate resource")
		certificate, err = getLetsEncryptCertificateByReference(o.Cxt, o.Reader, o.Req, o.LetsEncryptCertificate, o.NginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	// if CustomCertificate is provided, then find the certificate from CustomCertificate resource
	if o.CustomCertificate != nil {
		log.Info("CustomCertificate is provided, finding certificate from CustomCertificate resource")
		certificate, err = getCustomCertificateByReference(o.Cxt, o.Reader, o.Req, o.CustomCertificate, o.NginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	// if CertificateId is provided, then find the certificate from the ID
	if o.CertificateId != nil {
		log.Info("CertificateId is provided, finding certificate from ID")
		certificate, err = getCertificateFromID(o.Cxt, *o.CertificateId, o.NginxpmClient)
		if err != nil {
			return nil, err
		}
	}

	return certificate, nil
}

// Get certificate from Let's Encrypt reference
func getLetsEncryptCertificateByReference(ctx context.Context, r client.Reader, req ctrl.Request, reference *nginxpmoperatoriov1.SslLetsEncryptCertificate, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)
	lec := nginxpmoperatoriov1.LetsEncryptCertificate{}

	// If namespace is not provided, use the namespace of the proxyhost
	namespace := req.Namespace
	if reference.Namespace != nil {
		namespace = *reference.Namespace
	}

	// Retrieve the LetsEncryptCertificate resource
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reference.Name}, &lec); err != nil {
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
func getCustomCertificateByReference(ctx context.Context, r client.Reader, req ctrl.Request, reference *nginxpmoperatoriov1.SslCustomCertificate, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)
	customCert := nginxpmoperatoriov1.CustomCertificate{}

	// If namespace is not provided, use the namespace of the proxyhost
	namespace := req.Namespace
	if reference.Namespace != nil {
		namespace = *reference.Namespace
	}

	// Retrieve the CustomCertificate resource
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reference.Name}, &customCert); err != nil {
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
func getCertificateFromID(ctx context.Context, id int, nginxpmClient *nginxpm.Client) (*nginxpm.Certificate, error) {
	log := log.FromContext(ctx)

	certificate, err := nginxpmClient.FindCertificateByID(id)
	if err != nil {
		log.Error(err, "Failed to find certificate by ID")
		return nil, err
	}

	return certificate, nil
}
