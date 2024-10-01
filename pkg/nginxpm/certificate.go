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

package nginxpm

const (
	LETSENCRYPT_PROVIDER = "letsencrypt"
	CUSTOM_PROVIDER      = "other"
)

type certificate[K LetsEncryptCertificateMeta | CustomCertificateMeta] struct {
	ID          uint16   `json:"id"`
	CreatedOn   string   `json:"created_on"`
	ModifiedOn  string   `json:"modified_on"`
	Provider    string   `json:"provider"`
	NiceName    string   `json:"nice_name"`
	DomainNames []string `json:"domain_names"`
	ExpiresOn   string   `json:"expires_on"`
	Meta        K        `json:"meta"`
	Bound       bool     `json:"bound"`
}
