/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/libregraph/lico/identity"
	identityAuthorities "github.com/libregraph/lico/identity/authorities"
	identityClients "github.com/libregraph/lico/identity/clients"
	identityManagers "github.com/libregraph/lico/identity/managers"
	identityResources "github.com/libregraph/lico/identity/resources"
	"github.com/libregraph/lico/managers"
	codeManagers "github.com/libregraph/lico/oidc/code/managers"
)

type IdentityManagerFactory func(Bootstrap) (identity.Manager, error)

var identityManagerRegistry = make(map[string]IdentityManagerFactory)

func RegisterIdentityManager(name string, f IdentityManagerFactory) error {
	identityManagerRegistry[name] = f
	return nil
}

func getIdentityManagerByName(name string, bs Bootstrap) (identity.Manager, error) {
	if f, found := identityManagerRegistry[name]; !found {
		return nil, fmt.Errorf("no identity manager with name %s registered", name)
	} else {
		return f(bs)
	}
}

func newManagers(ctx context.Context, bs *bootstrap) (*managers.Managers, error) {
	logger := bs.config.Config.Logger

	var err error
	mgrs := managers.New()

	// Encryption manager.
	encryption, err := identityManagers.NewEncryptionManager(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption manager: %v", err)
	}

	err = encryption.SetKey(bs.config.EncryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid --encryption-secret parameter value for encryption: %v", err)
	}
	mgrs.Set("encryption", encryption)
	logger.Infof("encryption set up with %d key size", encryption.GetKeySize())

	// OIDC code manage.
	code := codeManagers.NewMemoryMapManager(ctx)
	mgrs.Set("code", code)

	// Identifier client registry manager.
	clients, err := identityClients.NewRegistry(ctx, bs.config.IssuerIdentifierURI, bs.config.IdentifierRegistrationConf, bs.config.Config.AllowDynamicClientRegistration, time.Duration(bs.config.DyamicClientSecretDurationSeconds)*time.Second, bs.config.Config.ClientIDMetadataDocumentHosts, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create client registry: %v", err)
	}
	mgrs.Set("clients", clients)

	// Identifier authorities registry manager.
	authorities, err := identityAuthorities.NewRegistry(ctx, bs.MakeURI(APITypeSignin, ""), bs.config.IdentifierAuthoritiesConf, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorities registry: %v", err)
	}
	mgrs.Set("authorities", authorities)

	// Protected resource registry manager.
	resources, err := identityResources.NewRegistry(ctx, bs.config.IdentifierRegistrationConf, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource registry: %v", err)
	}
	mgrs.Set("resources", resources)

	return mgrs, nil
}
