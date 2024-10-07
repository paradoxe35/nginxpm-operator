//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificate) DeepCopyInto(out *CustomCertificate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificate.
func (in *CustomCertificate) DeepCopy() *CustomCertificate {
	if in == nil {
		return nil
	}
	out := new(CustomCertificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CustomCertificate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificateCredentials) DeepCopyInto(out *CustomCertificateCredentials) {
	*out = *in
	out.Secret = in.Secret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificateCredentials.
func (in *CustomCertificateCredentials) DeepCopy() *CustomCertificateCredentials {
	if in == nil {
		return nil
	}
	out := new(CustomCertificateCredentials)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificateCredentialsSecret) DeepCopyInto(out *CustomCertificateCredentialsSecret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificateCredentialsSecret.
func (in *CustomCertificateCredentialsSecret) DeepCopy() *CustomCertificateCredentialsSecret {
	if in == nil {
		return nil
	}
	out := new(CustomCertificateCredentialsSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificateList) DeepCopyInto(out *CustomCertificateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CustomCertificate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificateList.
func (in *CustomCertificateList) DeepCopy() *CustomCertificateList {
	if in == nil {
		return nil
	}
	out := new(CustomCertificateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CustomCertificateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificateSpec) DeepCopyInto(out *CustomCertificateSpec) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(TokenName)
		(*in).DeepCopyInto(*out)
	}
	if in.NiceName != nil {
		in, out := &in.NiceName, &out.NiceName
		*out = new(string)
		**out = **in
	}
	out.Certificate = in.Certificate
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificateSpec.
func (in *CustomCertificateSpec) DeepCopy() *CustomCertificateSpec {
	if in == nil {
		return nil
	}
	out := new(CustomCertificateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomCertificateStatus) DeepCopyInto(out *CustomCertificateStatus) {
	*out = *in
	if in.Id != nil {
		in, out := &in.Id, &out.Id
		*out = new(int)
		**out = **in
	}
	if in.ExpiresOn != nil {
		in, out := &in.ExpiresOn, &out.ExpiresOn
		*out = new(string)
		**out = **in
	}
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomCertificateStatus.
func (in *CustomCertificateStatus) DeepCopy() *CustomCertificateStatus {
	if in == nil {
		return nil
	}
	out := new(CustomCertificateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomLocation) DeepCopyInto(out *CustomLocation) {
	*out = *in
	in.Forward.DeepCopyInto(&out.Forward)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomLocation.
func (in *CustomLocation) DeepCopy() *CustomLocation {
	if in == nil {
		return nil
	}
	out := new(CustomLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DnsChallenge) DeepCopyInto(out *DnsChallenge) {
	*out = *in
	out.ProviderCredentials = in.ProviderCredentials
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DnsChallenge.
func (in *DnsChallenge) DeepCopy() *DnsChallenge {
	if in == nil {
		return nil
	}
	out := new(DnsChallenge)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DnsChallengeProviderCredentials) DeepCopyInto(out *DnsChallengeProviderCredentials) {
	*out = *in
	out.Secret = in.Secret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DnsChallengeProviderCredentials.
func (in *DnsChallengeProviderCredentials) DeepCopy() *DnsChallengeProviderCredentials {
	if in == nil {
		return nil
	}
	out := new(DnsChallengeProviderCredentials)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DnsChallengeProviderCredentialsSecret) DeepCopyInto(out *DnsChallengeProviderCredentialsSecret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DnsChallengeProviderCredentialsSecret.
func (in *DnsChallengeProviderCredentialsSecret) DeepCopy() *DnsChallengeProviderCredentialsSecret {
	if in == nil {
		return nil
	}
	out := new(DnsChallengeProviderCredentialsSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Forward) DeepCopyInto(out *Forward) {
	*out = *in
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ForwardService)
		(*in).DeepCopyInto(*out)
	}
	if in.Host != nil {
		in, out := &in.Host, &out.Host
		*out = new(ForwardHost)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Forward.
func (in *Forward) DeepCopy() *Forward {
	if in == nil {
		return nil
	}
	out := new(Forward)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardHost) DeepCopyInto(out *ForwardHost) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardHost.
func (in *ForwardHost) DeepCopy() *ForwardHost {
	if in == nil {
		return nil
	}
	out := new(ForwardHost)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardService) DeepCopyInto(out *ForwardService) {
	*out = *in
	if in.Namespace != nil {
		in, out := &in.Namespace, &out.Namespace
		*out = new(string)
		**out = **in
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardService.
func (in *ForwardService) DeepCopy() *ForwardService {
	if in == nil {
		return nil
	}
	out := new(ForwardService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LetsEncryptCertificate) DeepCopyInto(out *LetsEncryptCertificate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LetsEncryptCertificate.
func (in *LetsEncryptCertificate) DeepCopy() *LetsEncryptCertificate {
	if in == nil {
		return nil
	}
	out := new(LetsEncryptCertificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LetsEncryptCertificate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LetsEncryptCertificateList) DeepCopyInto(out *LetsEncryptCertificateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LetsEncryptCertificate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LetsEncryptCertificateList.
func (in *LetsEncryptCertificateList) DeepCopy() *LetsEncryptCertificateList {
	if in == nil {
		return nil
	}
	out := new(LetsEncryptCertificateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LetsEncryptCertificateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LetsEncryptCertificateSpec) DeepCopyInto(out *LetsEncryptCertificateSpec) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(TokenName)
		(*in).DeepCopyInto(*out)
	}
	if in.DomainNames != nil {
		in, out := &in.DomainNames, &out.DomainNames
		*out = make([]DomainName, len(*in))
		copy(*out, *in)
	}
	if in.DnsChallenge != nil {
		in, out := &in.DnsChallenge, &out.DnsChallenge
		*out = new(DnsChallenge)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LetsEncryptCertificateSpec.
func (in *LetsEncryptCertificateSpec) DeepCopy() *LetsEncryptCertificateSpec {
	if in == nil {
		return nil
	}
	out := new(LetsEncryptCertificateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LetsEncryptCertificateStatus) DeepCopyInto(out *LetsEncryptCertificateStatus) {
	*out = *in
	if in.Id != nil {
		in, out := &in.Id, &out.Id
		*out = new(int)
		**out = **in
	}
	if in.DomainNames != nil {
		in, out := &in.DomainNames, &out.DomainNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExpiresOn != nil {
		in, out := &in.ExpiresOn, &out.ExpiresOn
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LetsEncryptCertificateStatus.
func (in *LetsEncryptCertificateStatus) DeepCopy() *LetsEncryptCertificateStatus {
	if in == nil {
		return nil
	}
	out := new(LetsEncryptCertificateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProxyHost) DeepCopyInto(out *ProxyHost) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProxyHost.
func (in *ProxyHost) DeepCopy() *ProxyHost {
	if in == nil {
		return nil
	}
	out := new(ProxyHost)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProxyHost) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProxyHostList) DeepCopyInto(out *ProxyHostList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ProxyHost, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProxyHostList.
func (in *ProxyHostList) DeepCopy() *ProxyHostList {
	if in == nil {
		return nil
	}
	out := new(ProxyHostList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProxyHostList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProxyHostSpec) DeepCopyInto(out *ProxyHostSpec) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(TokenName)
		(*in).DeepCopyInto(*out)
	}
	if in.DomainNames != nil {
		in, out := &in.DomainNames, &out.DomainNames
		*out = make([]DomainName, len(*in))
		copy(*out, *in)
	}
	if in.Ssl != nil {
		in, out := &in.Ssl, &out.Ssl
		*out = new(Ssl)
		(*in).DeepCopyInto(*out)
	}
	in.Forward.DeepCopyInto(&out.Forward)
	if in.CustomLocations != nil {
		in, out := &in.CustomLocations, &out.CustomLocations
		*out = make([]CustomLocation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProxyHostSpec.
func (in *ProxyHostSpec) DeepCopy() *ProxyHostSpec {
	if in == nil {
		return nil
	}
	out := new(ProxyHostSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProxyHostStatus) DeepCopyInto(out *ProxyHostStatus) {
	*out = *in
	if in.Id != nil {
		in, out := &in.Id, &out.Id
		*out = new(int)
		**out = **in
	}
	if in.CertificateId != nil {
		in, out := &in.CertificateId, &out.CertificateId
		*out = new(int)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProxyHostStatus.
func (in *ProxyHostStatus) DeepCopy() *ProxyHostStatus {
	if in == nil {
		return nil
	}
	out := new(ProxyHostStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Secret) DeepCopyInto(out *Secret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Secret.
func (in *Secret) DeepCopy() *Secret {
	if in == nil {
		return nil
	}
	out := new(Secret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretData) DeepCopyInto(out *SecretData) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretData.
func (in *SecretData) DeepCopy() *SecretData {
	if in == nil {
		return nil
	}
	out := new(SecretData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Ssl) DeepCopyInto(out *Ssl) {
	*out = *in
	if in.LetsEncryptCertificate != nil {
		in, out := &in.LetsEncryptCertificate, &out.LetsEncryptCertificate
		*out = new(SslLetsEncryptCertificate)
		(*in).DeepCopyInto(*out)
	}
	if in.CustomCertificate != nil {
		in, out := &in.CustomCertificate, &out.CustomCertificate
		*out = new(SslCustomCertificate)
		(*in).DeepCopyInto(*out)
	}
	if in.CertificateId != nil {
		in, out := &in.CertificateId, &out.CertificateId
		*out = new(int)
		**out = **in
	}
	if in.LetsEncryptEmail != nil {
		in, out := &in.LetsEncryptEmail, &out.LetsEncryptEmail
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Ssl.
func (in *Ssl) DeepCopy() *Ssl {
	if in == nil {
		return nil
	}
	out := new(Ssl)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SslCustomCertificate) DeepCopyInto(out *SslCustomCertificate) {
	*out = *in
	if in.Namespace != nil {
		in, out := &in.Namespace, &out.Namespace
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SslCustomCertificate.
func (in *SslCustomCertificate) DeepCopy() *SslCustomCertificate {
	if in == nil {
		return nil
	}
	out := new(SslCustomCertificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SslLetsEncryptCertificate) DeepCopyInto(out *SslLetsEncryptCertificate) {
	*out = *in
	if in.Namespace != nil {
		in, out := &in.Namespace, &out.Namespace
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SslLetsEncryptCertificate.
func (in *SslLetsEncryptCertificate) DeepCopy() *SslLetsEncryptCertificate {
	if in == nil {
		return nil
	}
	out := new(SslLetsEncryptCertificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Token) DeepCopyInto(out *Token) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Token.
func (in *Token) DeepCopy() *Token {
	if in == nil {
		return nil
	}
	out := new(Token)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Token) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenList) DeepCopyInto(out *TokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Token, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenList.
func (in *TokenList) DeepCopy() *TokenList {
	if in == nil {
		return nil
	}
	out := new(TokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenName) DeepCopyInto(out *TokenName) {
	*out = *in
	if in.Namespace != nil {
		in, out := &in.Namespace, &out.Namespace
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenName.
func (in *TokenName) DeepCopy() *TokenName {
	if in == nil {
		return nil
	}
	out := new(TokenName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenSpec) DeepCopyInto(out *TokenSpec) {
	*out = *in
	out.Secret = in.Secret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenSpec.
func (in *TokenSpec) DeepCopy() *TokenSpec {
	if in == nil {
		return nil
	}
	out := new(TokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenStatus) DeepCopyInto(out *TokenStatus) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(string)
		**out = **in
	}
	if in.Expires != nil {
		in, out := &in.Expires, &out.Expires
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenStatus.
func (in *TokenStatus) DeepCopy() *TokenStatus {
	if in == nil {
		return nil
	}
	out := new(TokenStatus)
	in.DeepCopyInto(out)
	return out
}
