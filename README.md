# Nginx Proxy Manager Operator

The Nginx Proxy Manager Operator is a Kubernetes operator built using Kubebuilder. It simplifies the management of Nginx Proxy Manager resources by representing them as Kubernetes objects. This operator provides seamless integration between your Kubernetes cluster and Nginx Proxy Manager, streamlining the process of managing your HTTP redirections, SSL configurations, and other proxy-related tasks within your Kubernetes environment.

This solution is particularly beneficial for homelab setups and environments where Kubernetes is used alongside tools like OPNsense for firewall management. If you’re using Nginx Proxy Manager for HTTP redirection to services behind your firewall and have a Kubernetes cluster in your network, the operator helps automate the management of Nginx configurations from within Kubernetes.

> If you do not have a load balancer service set up in your Kubernetes cluster, we recommend using a `NodePort` service type along with a [forked](https://github.com/paradoxe35/nginx-proxy-manager) version of Nginx Proxy Manager that supports an Nginx load balancer (this [fork](https://github.com/paradoxe35/nginx-proxy-manager) is always kept up to date with the upstream repository). When using the NodePort service type, this operator automatically gathers the host IPs of all service pods and configures them as upstreams in the Nginx load balancer, making the process of scaling and managing services easier and more efficient.

## Features

| Feature                     | Status                 |
| --------------------------- | ---------------------- |
| Token (create access token) | ✅ Implemented         |
| Let's Encrypt Certificate   | ✅ Implemented         |
| Custom Certificate          | ✅ Implemented         |
| Proxy Host                  | ✅ Implemented         |
| Access Lists                | ✅ Implemented         |
| Streams                     | ✅ Implemented         |
| Redirection Hosts           | ❌ Not yet implemented |
| 404 Hosts                   | ❌ Not yet implemented |

## Installation

To install the operator, run the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/paradoxe35/nginxpm-operator/v0.2.5/dist/install.yaml
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
    # Or you can use hosts configuration
    hosts:
      - hostName: 192.168.1.4 # HostName|IP
        hostPort: 80

  # Uncomment and modify the following sections as needed
  # bindExisting: true
  # blockExploits: true
  # websocketSupport: true
  # cachingEnabled: false

  # Access List
  # accessList:
  #   name: admin-access # Access list resource name
  #   namespace: default # Access list resource namespace
  #   accessListId: 1 # if you know the accessList id of an existing accessList in the nginx-proxy-manager instance (optional)

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
  #       # Or you can use the hosts configuration
  #       hosts:
  #         - hostName: 192.168.1.4 # HostName|IP
  #           hostPort: 80
```

Apply these resources to your Kubernetes cluster:

```bash
kubectl apply -f token.yaml
kubectl apply -f proxy-host.yaml
```

If all the information is correct, you should see a Proxy Host created with the specified domains in your Nginx Proxy Manager instance.

### 3. Create a Stream

Or create a Stream. Save the following YAML as `stream.yaml`:

```yaml
apiVersion: nginxpm-operator.io/v1
kind: Stream
metadata:
  labels:
    app.kubernetes.io/name: nginxpm-operator
  name: stream-sample
spec:
  # Token is optional, if not provided, the operator will try to find a token with `token-nginxpm` name
  # in the same namespace as the proxyhost is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
  token:
    name: token-nginxpm
    namespace: default

  incomingPort: 3000

  forward:
    tcpForwarding: true
    udpForwarding: false
    # service:
    #   name: nginx-service
    #   namespace: default
    # Or you can use the hosts configuration
    hosts:
      - hostName: 192.168.1.4 # HostName|IP
        hostPort: 80

  # overwriteIncomingPortWithForwardPort: false

  # Enable ssl
  # ssl:
  #   certificateId: 1 # if you know the certificate id of an existing certificate in the nginx-proxy-manager instance

  #   customCertificate:
  #     name: custom-certificate-sample
  #     namespace: nginxpm-operator-system

  #   letsEncryptCertificate:
  #     name: letsencrypt-certificate-sample
  #     namespace: nginxpm-operator-system
```

Apply these resources to your Kubernetes cluster:

```bash
kubectl apply -f stream.yaml
```

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

Attach this to your `ProxyHost` or `Stream` using `ssl.letsEncryptCertificate.name` in the spec.

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

Attach this to your `ProxyHost` or `Stream` using `ssl.customCertificate.name` in the spec.

## AccessList

```yaml
apiVersion: nginxpm-operator.io/v1
kind: AccessList
metadata:
  labels:
    app.kubernetes.io/name: nginxpm-operator
  name: accesslist-sample
spec:
  # Token is optional, if not provided, the operator will try to find a token with `token-nginxpm` name
  # in the same namespace as the letsencryptcertificate is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
  token:
    name: token-sample
    namespace: default

  name: Admin Access

  satisfyAny: true
  passAuth: false

  authorizations:
    - username: admin
      password: password

  clients:
    - address: 192.168.11.2/24
      directive: allow
```

Attach this to your `ProxyHost` using `accessList.name` in the spec.

## Support

If you find this tool helpful for your setup, similar to the author's use case, please consider starring the repository or contributing to the source code.
