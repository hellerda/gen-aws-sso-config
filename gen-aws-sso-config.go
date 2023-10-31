// -------------------------------------------------------------------------------------------------
// Auto generate a user's AWS config file for SSO login through Identity Center.
//
// The program will open a browser window for you to SSO login.  Afterward, close the tab and hit
// return.  The program will build your config and output it to stdout.  To use, add the snip
// to your ".aws/config" file.
//
// Based on: https://github.com/aws/aws-sdk-go-v2/issues/1222
// -------------------------------------------------------------------------------------------------

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/pkg/browser"
)

var (
	startURL       string
	ssoSessionName string
	ssoRegion      string
)

func main() {

	flag.StringVar(&startURL, "start-url", "", "AWS SSO Start URL")
	flag.StringVar(&ssoRegion, "sso-region", "", "AWS IdC instance region")
	flag.StringVar(&ssoSessionName, "sso-session-name", "my-sso", "The sso_session identifier to use in your config file")
	flag.Parse()

	if startURL == "" || ssoRegion == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Load default aws config...
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(ssoRegion))
	if err != nil {
		fmt.Println(err)
	}

	// Create new SSO OIDC client to trigger login flow...
	ssooidcClient := ssooidc.NewFromConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}

	// Register the OIDC client...
	register, err := ssooidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("sample-client-name"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		fmt.Println(err)
	}

	// Authorize your device using the client registration response...
	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		fmt.Println(err)
	}

	// Open browser window for OIDC login, wait for user to press enter to continue...
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Printf("\nIf browser is not opened automatically, please open link:\n%v\n", url)
	err = browser.OpenURL(url)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Press ENTER key once login is done")
	_ = bufio.NewScanner(os.Stdin).Scan()

	// Fetch SSO access token...
	token, err := ssooidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		DeviceCode:   deviceAuth.DeviceCode,
		GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
	})
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("======== ADD THE FOLLOWING TO YOUR .aws/config FILE ========\n")

	// Create the IdC portal entry...
	fmt.Printf("\n")
	fmt.Printf("# This is the Identity Center portal entry\n")
	fmt.Printf("[sso-session %v]\n", ssoSessionName)
	fmt.Printf("sso_region = %v\n", ssoRegion)
	fmt.Printf("sso_start_url = %v\n", startURL)
	fmt.Printf("# sso_registration_scopes = sso:account:access\n")
	fmt.Printf("\n")

	// Create a profile entry for each account+role we find for this user...
	ssoClient := sso.NewFromConfig(cfg)

	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token.AccessToken,
	})
	for accountPaginator.HasMorePages() {
		x, err := accountPaginator.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)
		}
		for _, y := range x.AccountList {

			rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
				AccessToken: token.AccessToken,
				AccountId:   y.AccountId,
			})
			for rolePaginator.HasMorePages() {
				z, err := rolePaginator.NextPage(context.TODO())
				if err != nil {
					fmt.Println(err)
				}
				for _, p := range z.RoleList {
					fmt.Printf("[profile %v-%v]\n", aws.ToString(y.AccountName), aws.ToString(p.RoleName))
					fmt.Printf("sso_session = %v\n", ssoSessionName)
					fmt.Printf("sso_account_id = %v\n", aws.ToString(p.AccountId))
					fmt.Printf("sso_role_name = %v\n", aws.ToString(p.RoleName))
					fmt.Printf("# region = %v\n", ssoRegion)
					fmt.Printf("\n")
				}
			}
		}
	}
}
