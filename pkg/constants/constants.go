/*
Copyright © 2018 inwinSTACK Inc

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
/*
Copyright © 2018 inwinSTACK Inc

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

package constants

const Finalizer = "kubernetes"

const PolicyPrefix = "k8s"

// Annotation Keys
const (
	// AllowSecurityKey is the key of annotation for enabling security policy
	AllowSecurityKey = "inwinstack.com/allow-security-policy"
	// AllowNATKey is the key of annotation for enabling nat policy
	AllowNATKey = "inwinstack.com/allow-nat-policy"
	// ExternalPoolKey is the key of annotation for assigning IP from which pool
	ExternalPoolKey = "inwinstack.com/external-pool"
	// PublicIPKey is the key of annotation for recording IP
	PublicIPKey = "inwinstack.com/allocated-public-ip"
	// ServiceRefreshKey is the key of annotation for refreshing Kubernetes service object
	ServiceRefreshKey = "inwinstack.com/service-refresh"
	// WhiteListAddressesKey is the key of annotations for the whitelist
	WhiteListAddressesKey = "inwinstack.com/whitelist-addresses"
)
