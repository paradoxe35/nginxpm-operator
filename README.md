# Nginx Proxy Manager Operator

Nginx Proxy Manager Operator is a Kubernetes operator built with kubebuilder. It represents Nginx Proxy Manager resources as Kubernetes objects, providing a seamless integration between your Kubernetes cluster and Nginx Proxy Manager.

This solution is particularly useful for homelab setups. If you're using Nginx Proxy Manager for HTTP redirection to services behind your firewall (e.g., OPNsense) and have a Kubernetes cluster within your network, this operator simplifies the process of managing Nginx Proxy Manager resources through Kubernetes.

Instead of manually configuring redirects from Nginx Proxy Manager to your ingress controller or services.

## Features

| Feature                     | Status                 |
| --------------------------- | ---------------------- |
| Token (create access token) | ✅ Implemented         |
| Let's Encrypt Certificate   | ✅ Implemented         |
| Custom Certificate          | ✅ Implemented         |
| Proxy Host                  | ✅ Implemented         |
| Redirection Hosts           | ❌ Not yet implemented |
| Streams                     | ❌ Not yet implemented |
| 404 Hosts                   | ❌ Not yet implemented |
| Access Lists                | ❌ Not yet implemented |

## Installation

To install the operator, run the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/paradoxe35/nginxpm-operator/main/dist/install.yaml
```

## Quick Start Guide

### 1. Create a Token Resource

First, create a Token resource. Save the following YAML as `token.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nginxpm-secret
  namespace: default
type: Opaque
data:
  identity: YWRtaW5AZXhhbXBsZS5jb20= # admin@example.com
  secret: Y2hhbmdlbWU= # changeme

---
apiVersion: nginxpm-operator.io/v1
kind: Token
metadata:
  name: token-nginxpm
  namespace: default
  labels:
    app.kubernetes.io/name: nginxpm-operator
spec:
  endpoint: http://[IP|DOMAIN]:81
  secret:
    secretName: nginxpm-secret
```

### 2. Create a Proxy Host

Next, create a Proxy Host. Save the following YAML as `proxy-host.yaml`:

```yaml
apiVersion: nginxpm-operator.io/v1
kind: ProxyHost
metadata:
  labels:
    app.kubernetes.io/name: nginxpm-operator
  name: proxyhost-example
spec:
  # Token is optional, if not provided, the operator will try to find a token with `token-nginxpm` name
  # in the same namespace as the proxyhost is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
  token:
    name: token-nginxpm
    namespace: default

  domainNames:
    - example.com

  forward:
    scheme: http
    # service:
    #   name: nginx-svc
    #   namespace: default
    # Or you can use the host configuration
    host:
      hostName: 192.168.1.4 # HostName|IP
      hostPort: 80

  # Uncomment and modify the following sections as needed
  # blockExploits: true
  # websocketSupport: true
  # cachingEnabled: false

  # Enable ssl here
  # ssl:
  #   autoCertificateRequest: true
  #   sslForced: true
  #   http2Support: true
  #   letsEncryptEmail: user@example.com

  #   certificateId: 1 # if you know the certificate id of an existing certificate in the nginx-proxy-manager instance
  #   customCertificate:
  #     name: custom-certificate-sample

  #   letsEncryptCertificate:
  #     name: letsencrypt-certificate-sample

  #   sslForced: true
  #   http2Support: true
  #   hstsEnabled: false # More info https://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
  #   hstsSubdomains: false

  # customLocations:
  #   - locationPath: /
  #     forward:
  #       scheme: http
  #       path: /hello
  #       # service:
  #       #   name: nginx-svc
  #       #   namespace: default
  #       # Or you can use the host configuration
  #       host:
  #         hostName: 192.168.1.4 # HostName|IP
  #         hostPort: 80
```

### 3. Apply the Resources

Apply these resources to your Kubernetes cluster:

```bash
kubectl apply -f token.yaml
kubectl apply -f proxy-host.yaml
```

If all the information is correct, you should see a Proxy Host created with the specified domains in your Nginx Proxy Manager instance.

## Certificates

You can generate or attach certificates automatically with the ProxyHost spec using `ssl.autoCertificateRequest: true`. For more granular control, you can use the following certificate objects:

### 1. LetsEncryptCertificate

```yaml
apiVersion: nginxpm-operator.io/v1
kind: LetsEncryptCertificate
metadata:
  labels:
    app.kubernetes.io/name: nginxpm-operator
  name: letsencryptcertificate-sample
spec:
  # Token is optional, if not provided, the operator will try to find a token with `token-nginxpm` name
  # in the same namespace as the letsencryptcertificate is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
  token:
    name: token-nginxpm
    namespace: default

  domainNames:
    - example.com
    - www.example.com

  letsEncryptEmail: example@example.com

  dnsChallenge: # Optional
    provider: acmedns
    providerCredentials:
      secret:
        name: dns-credentials-sample

---
apiVersion: v1
kind: Secret
metadata:
  name: dns-credentials-sample
type: Opaque
data:
  credentials: YWRtaW4=
```

Attach this to your ProxyHost using `ssl.letsEncryptCertificate.name` in the spec.

### 2. CustomCertificate

```yaml
apiVersion: nginxpm-operator.io/v1
kind: CustomCertificate
metadata:
  labels:
    app.kubernetes.io/name: nginxpm-operator
  name: customcertificate-sample
spec:
  # Token is optional, if not provided, the operator will try to find a token with `token-nginxpm` name
  # in the same namespace as the letsencryptcertificate is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
  token:
    name: token-nginxpm
    namespace: default

  niceName: example-certificate # Optional

  certificate:
    secret:
      name: certificate-sample

---
apiVersion: v1
kind: Secret
metadata:
  name: certificate-sample
type: Opaque
data:
  certificate: YWRtaW4=
  certificate_key: YWRtaW4=
```

Attach this to your ProxyHost using `ssl.customCertificate.name` in the spec.


## Support

If you find this tool helpful for your setup, similar to the author's use case, please consider starring the repository or contributing to the source code.
