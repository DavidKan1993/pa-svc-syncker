/*
Copyright © 2018 inwinSTACK.inc
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

package k8sutil

import (
	"testing"

	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
)

func TestIP(t *testing.T) {
	client := fake.NewSimpleClientset()

	_, err := CreateIP(client, "test-ip", "default", "test-pool")
	assert.Nil(t, err)

	ip, err := GetIP(client, "test-ip", "default")
	assert.Nil(t, err)
	assert.NotNil(t, ip)

	assert.Nil(t, DeleteIP(client, "test-ip", "default"))
}
