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

package proxyhost

import (
	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
)

// CaptureInitialConfiguration captures the initial state of an existing NPM proxy host
// before any modifications are made by the operator
func CaptureInitialConfiguration(proxyHost *nginxpm.ProxyHost) *nginxpmoperatoriov1.InitialConfiguration {
	if proxyHost == nil {
		return nil
	}

	config := &nginxpmoperatoriov1.InitialConfiguration{
		DomainNames:           proxyHost.DomainNames,
		ForwardHost:           proxyHost.ForwardHost,
		ForwardPort:           proxyHost.ForwardPort,
		ForwardScheme:         proxyHost.ForwardScheme,
		AccessListId:          proxyHost.AccessListID,
		SSLForced:             proxyHost.SSLForced,
		CachingEnabled:        proxyHost.CachingEnabled,
		BlockExploits:         proxyHost.BlockExploits,
		AllowWebsocketUpgrade: proxyHost.AllowWebsocketUpgrade,
		HTTP2Support:          proxyHost.HTTP2Support,
		HSTSEnabled:           proxyHost.HSTSEnabled,
		HSTSSubdomains:        proxyHost.HSTSSubdomains,
		AdvancedConfig:        proxyHost.AdvancedConfig,
		Enabled:               proxyHost.Enabled,
	}

	// Capture certificate ID if present
	if proxyHost.CertificateID != 0 {
		certID := proxyHost.CertificateID
		config.CertificateId = &certID
	}

	// Capture locations if present
	if len(proxyHost.Locations) > 0 {
		config.Locations = make([]nginxpmoperatoriov1.ProxyHostLocationConfig, len(proxyHost.Locations))
		for i, loc := range proxyHost.Locations {
			config.Locations[i] = nginxpmoperatoriov1.ProxyHostLocationConfig{
				Path:           loc.Path,
				AdvancedConfig: loc.AdvancedConfig,
				ForwardScheme:  loc.ForwardScheme,
				ForwardHost:    loc.ForwardHost,
				ForwardPort:    loc.ForwardPort,
			}
		}
	}

	return config
}

// BuildRestorationInput creates a ProxyHostRequestInput from the stored initial configuration
// to restore the original settings when a bound resource is deleted
func BuildRestorationInput(config *nginxpmoperatoriov1.InitialConfiguration) *nginxpm.ProxyHostRequestInput {
	if config == nil {
		return nil
	}

	input := &nginxpm.ProxyHostRequestInput{
		DomainNames:           config.DomainNames,
		ForwardHost:           config.ForwardHost,
		ForwardScheme:         config.ForwardScheme,
		ForwardPort:           config.ForwardPort,
		AdvancedConfig:        config.AdvancedConfig,
		BlockExploits:         config.BlockExploits,
		AllowWebsocketUpgrade: config.AllowWebsocketUpgrade,
		CachingEnabled:        config.CachingEnabled,
		AccessListID:          config.AccessListId,
		CertificateID:         config.CertificateId,
		SSLForced:             config.SSLForced,
		HTTP2Support:          config.HTTP2Support,
		HSTSEnabled:           config.HSTSEnabled,
		HSTSSubdomains:        config.HSTSSubdomains,
		CustomFields:          make(nginxpm.RequestCustomFields),
		Locations:             []nginxpm.ProxyHostLocation{},
	}

	// Restore locations if present
	if len(config.Locations) > 0 {
		input.Locations = make([]nginxpm.ProxyHostLocation, len(config.Locations))
		for i, loc := range config.Locations {
			input.Locations[i] = nginxpm.ProxyHostLocation{
				Path:           loc.Path,
				AdvancedConfig: loc.AdvancedConfig,
				ForwardScheme:  loc.ForwardScheme,
				ForwardHost:    loc.ForwardHost,
				ForwardPort:    loc.ForwardPort,
			}
		}
	}

	return input
}

// ShouldCaptureInitialConfig determines if we should capture the initial configuration
// This happens when binding to an existing proxy host for the first time
func ShouldCaptureInitialConfig(ph *nginxpmoperatoriov1.ProxyHost, proxyHost *nginxpm.ProxyHost) bool {
	// Capture if:
	// 1. We're binding to an existing proxy host (BindExisting is true)
	// 2. We found an existing proxy host
	// 3. We haven't captured the initial config yet
	return ph.Spec.BindExisting &&
		proxyHost != nil &&
		ph.Status.InitialConfiguration == nil
}

// ShouldRestoreInitialConfig determines if we should restore the initial configuration
// This happens when deleting a bound resource that has stored initial configuration
func ShouldRestoreInitialConfig(ph *nginxpmoperatoriov1.ProxyHost) bool {
	return ph.Status.Bound &&
		ph.Status.InitialConfiguration != nil
}
