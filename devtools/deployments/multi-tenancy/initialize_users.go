package main

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-resty/resty/v2"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

const (
	provisioningAPIURL    = "http://localhost:9120/graph"
	provisioningAuthToken = "changeme"
)

type tenantWithUsers struct {
	tenant libregraph.EducationSchool
	users  []libregraph.EducationUser
}

var demoTenants = []tenantWithUsers{
	{
		tenant: libregraph.EducationSchool{
			DisplayName: libregraph.PtrString("Famous Coders"),
			ExternalId:  libregraph.PtrString("famouscodersExternalID"),
		},
		users: []libregraph.EducationUser{
			{
				DisplayName:              libregraph.PtrString("Dennis Ritchie"),
				OnPremisesSamAccountName: libregraph.PtrString("dennis"),
				Mail:                     libregraph.PtrString("dennis@example.org"),
			},
			{
				DisplayName:              libregraph.PtrString("Grace Hopper"),
				OnPremisesSamAccountName: libregraph.PtrString("grace"),
				Mail:                     libregraph.PtrString("grace@example.org"),
			},
		},
	},
	{
		tenant: libregraph.EducationSchool{
			DisplayName: libregraph.PtrString("Scientists"),
			ExternalId:  libregraph.PtrString("scientistsExternalID"),
		},
		users: []libregraph.EducationUser{
			{
				DisplayName:              libregraph.PtrString("Albert Einstein"),
				OnPremisesSamAccountName: libregraph.PtrString("einstein"),
				Mail:                     libregraph.PtrString("einstein@example.org"),
			},
			{
				DisplayName:              libregraph.PtrString("Marie Curie"),
				OnPremisesSamAccountName: libregraph.PtrString("marie"),
				Mail:                     libregraph.PtrString("marie@example.org"),
			},
		},
	},
}

func main() {
	lgconf := libregraph.NewConfiguration()
	lgconf.Servers = libregraph.ServerConfigurations{
		{
			URL: provisioningAPIURL,
		},
	}
	lgconf.DefaultHeader = map[string]string{"Authorization": "Bearer " + provisioningAuthToken}
	lgclient := libregraph.NewAPIClient(lgconf)

	for _, tenant := range demoTenants {
		tenantid, err := createTenant(lgclient, tenant.tenant)
		if err != nil {
			fmt.Printf("Failed to create tenant: %s\n", err)
			continue
		}
		for _, user := range tenant.users {
			userid1, err := createUser(lgclient, user)
			if err != nil {
				fmt.Printf("Failed to create user: %s\n", err)
				continue
			}
			_, err = lgclient.EducationSchoolApi.AddUserToSchool(context.TODO(), tenantid).EducationUserReference(libregraph.EducationUserReference{
				OdataId: libregraph.PtrString(fmt.Sprintf("%s/education/users/%s", provisioningAPIURL, userid1)),
			}).Execute()
			if err != nil {
				fmt.Printf("Failed to add user to tenant: %s\n", err)
				continue
			}
		}
	}

	resetAllUserPasswords()
	setUserRoles()
}

func createUser(client *libregraph.APIClient, user libregraph.EducationUser) (string, error) {
	newUser, _, err := client.EducationUserApi.CreateEducationUser(context.TODO()).EducationUser(user).Execute()
	if err != nil {
		fmt.Printf("Failed to create user: %s\n", err)
		return "", err
	}
	fmt.Printf("Created user: %s with id %s\n", newUser.GetDisplayName(), newUser.GetId())
	return newUser.GetId(), nil
}

func createTenant(client *libregraph.APIClient, tenant libregraph.EducationSchool) (string, error) {
	newTenant, _, err := client.EducationSchoolApi.CreateSchool(context.TODO()).EducationSchool(tenant).Execute()
	if err != nil {
		fmt.Printf("Failed to create tenant: %s\n", err)
		return "", err
	}
	fmt.Printf("Created tenant: %s with id %s\n", newTenant.GetDisplayName(), newTenant.GetId())
	return newTenant.GetId(), nil
}

func setUserRoles() {
	tls := tls.Config{InsecureSkipVerify: true}
	restyClient := resty.New().SetTLSClientConfig(&tls)
	client := gocloak.NewClient("https://keycloak.opencloud.test")
	client.SetRestyClient(restyClient)
	ctx := context.Background()
	token, err := client.LoginAdmin(ctx, "kcadmin", "admin", "master")
	if err != nil {
		fmt.Printf("Failed to login: %s\n", err)
		panic("Something wrong with the credentials or url")
	}

	role, _ := client.GetRealmRole(ctx, token.AccessToken, "openCloud", "opencloudUser")
	users, err := client.GetUsers(ctx, token.AccessToken, "openCloud", gocloak.GetUsersParams{})
	if err != nil {
		fmt.Printf("%s\n", err)
		panic("Oh no!, failed to list users :(")
	}
	for _, user := range users {
		err := client.AddRealmRoleToUser(ctx, token.AccessToken, "openCloud", *user.ID, []gocloak.Role{
			*role,
		})
		if err != nil {
			fmt.Printf("Failed to assign role to user %s: %s\n", *user.Username, err)
		}
	}
}

func resetAllUserPasswords() {
	tls := tls.Config{InsecureSkipVerify: true}
	restyClient := resty.New().SetTLSClientConfig(&tls)
	client := gocloak.NewClient("https://keycloak.opencloud.test")
	client.SetRestyClient(restyClient)
	ctx := context.Background()
	token, err := client.LoginAdmin(ctx, "kcadmin", "admin", "master")
	if err != nil {
		fmt.Printf("Failed to login: %s\n", err)
		panic("Something wrong with the credentials or url")
	}

	users, err := client.GetUsers(ctx, token.AccessToken, "openCloud", gocloak.GetUsersParams{})
	if err != nil {
		fmt.Printf("%s\n", err)
		panic("Oh no!, failed to list users :(")
	}
	for _, user := range users {
		fmt.Printf("Setting password for user: %s\n", *user.Username)
		err = client.SetPassword(ctx, token.AccessToken, *user.ID, "openCloud", "demo", false)
		if err != nil {
			fmt.Printf("Failed to set password for user %s: %s\n", *user.Username, err)
		}
	}

}
