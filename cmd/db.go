package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"terra3-cli/ssmclient"
	"time"

	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/aws/smithy-go"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pkg/browser"
)

func init() {
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(loginCmd)
	dbCmd.AddCommand(dbPortForwardCmd)
	dbPortForwardCmd.Flags().StringVarP(&profile, "profile", "p", "", "Optional AWS profile to use. If not provided, a selection menu will open.")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Built-in OIDC login without requiring the AWS CLI. (experimental)",
	Long:  `Built-in OIDC login without requiring the AWS CLI. This feature is experimental.`,
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Interact with your provisioned AWS RDS instance.",
	Long: `Interact with your provisioned AWS RDS instance. Use one of the sub-commands.
	* port-forward: Establish a port forwarding to your RDS instance.
	`,
}

var dbPortForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Create a secure port-forward to the private RDS database using SSM.",
	Long: `Create a secure port-forward to the private RDS database using SSM. If used without the profile parameter,
	it will open up a selection menu to choose the AWS profile to use. If used with a profile parameter, it will use 
	the given profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		dbPortForwardToDB()
	},
}

func dbPortForwardToDB() {
	fmt.Print("Terra3 CLI: Establish a secure port-forward to the private RDS database using SSM with the profile you are going to pick.\n\n")

	// If profile is given by dbPortForwardCmd.Flags() then set os.Setenv("AWS_PROFILE", result)
	if profile != "" {
		os.Setenv("AWS_PROFILE", profile)
	} else {
		// Load all AWS profiles
		profiles, err := loadAllAWSProfiles()
		if err != nil {
			log.Fatalf("unable to load AWS profiles, %v", err)
		}

		// Prompt user to select a profile
		prompt := promptui.Select{
			Label:     "Select AWS profile",
			Items:     profiles,
			CursorPos: 1,
		}

		_, result, err := prompt.Run()
		if err != nil {
			log.Fatalf("prompt failed %v", err)
		}

		// Set the selected profile as the default profile
		os.Setenv("AWS_PROFILE", result)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	rdsClient := rds.NewFromConfig(cfg)

	bastionHostID, err := getBastionHostID(ec2Client)
	if err != nil {
		log.Fatalf("unable to get bastion host ID, %v", err)
	}

	rdsURL, err := getRDSURL(rdsClient)
	if err != nil {
		log.Fatalf("unable to get RDS URL, %v", err)
	}

	showIamDetails()

	fmt.Printf("\nBastion host found with id: %s\n", bastionHostID)
	fmt.Printf("RDS database found with url: %s:%d\n", rdsURL, 3306)

	wordPromptContent := promptContent{
		"Please provide a port number.",
		"What port number would you like to be opened locally?",
	}
	inputLocalPort := promptGetInput(wordPromptContent)

	// Convert inputLocalPort from string to int
	localPort, err := strconv.Atoi(inputLocalPort)
	if err != nil {
		log.Fatal(err)
	}

	// create ssm tunnel with internal ssh
	ssm_tunnel(bastionHostID, rdsURL, localPort)
}

func getBastionHostID(client *ec2.Client) (string, error) {
	resp, err := client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{})
	if err != nil {
		return "", err
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			for _, tag := range instance.Tags {
				if strings.Contains(*tag.Value, "bastion") {
					return *instance.InstanceId, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no bastion host found")
}

func getRDSURL(client *rds.Client) (string, error) {
	resp, err := client.DescribeDBInstances(context.TODO(), &rds.DescribeDBInstancesInput{})
	if err != nil {
		return "", err
	}

	if len(resp.DBInstances) == 0 {
		return "", fmt.Errorf("no RDS instances found")
	}

	return *resp.DBInstances[0].Endpoint.Address, nil
}

// add array of constants containing all AWS regions available
var awsRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"ap-east-1",
	"ap-south-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-northeast-3",
	"ap-southeast-1",
	"ap-southeast-2",
	"ca-central-1",
	"cn-north-1",
	"cn-northwest-1",
	"eu-central-1",
	"eu-central-2",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"il-central-1",
	"me-south-1",
	"sa-east-1",
	"me-central-1",
	"us-gov-east-1",
	"us-gov-west-1",
}

func login() {

	// prompt user for a url
	// Prompt user to select a profile
	prompt1 := promptui.Prompt{
		Label:   "SSO URL",
		Default: "",
	}

	resultSSOUrl, err := prompt1.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	// Prompt user to select a profile
	prompt := promptui.Select{
		Label:     "Please provide an AWS region.",
		Items:     awsRegions,
		CursorPos: 14,
	}
	_, resultRegion, err := prompt.Run()
	if err != nil {
		log.Fatalf("prompt failed %v", err)
	}

	var (
		startURL string = "https://" + resultSSOUrl + ".awsapps.com/start"
		region          = resultRegion
	)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithDefaultRegion(region))
	if err != nil {
		log.Fatalf("%v", err)
	}
	// create SSO oidcClient client to trigger login flow
	oidcClient := ssooidc.NewFromConfig(cfg)

	// register your client which is triggering the login flow
	register, err := oidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("terra3-cli-client"),
		ClientType: aws.String("public"),
	})

	if err != nil {
		log.Fatal(err)
	}

	// authorize your device using the client registration response
	deviceAuth, err := oidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(startURL),
	})

	if err != nil {
		log.Fatal(err)
	}

	url := aws.ToString(deviceAuth.VerificationUriComplete)
	log.Printf("If your browser is not opened automatically, please open link:\n%v\n", url)
	err = browser.OpenURL(url)
	if err != nil {
		log.Fatal(err)
	}

	var token *ssooidc.CreateTokenOutput
	approved := false

	// poll the client until it has finished authorization.
	for !approved {
		t, err := oidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
			ClientId:     register.ClientId,
			ClientSecret: register.ClientSecret,
			DeviceCode:   deviceAuth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err != nil {
			isPending := strings.Contains(err.Error(), "AuthorizationPendingException:")
			log.Println("Authorization pending...")
			if isPending {
				log.Print(".")
				time.Sleep(time.Duration(deviceAuth.Interval) * time.Second)
				continue
			}
		}
		approved = true
		token = t
	}

	ssoClient := sso.NewFromConfig(cfg)

	log.Println("Fetching list of accounts for this user")
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token.AccessToken,
	})

	for accountPaginator.HasMorePages() {
		x, err := accountPaginator.NextPage(context.TODO())
		if err != nil {
			log.Fatal(err)
		}
		for _, y := range x.AccountList {
			log.Println("-------------------------------------------------------")
			log.Printf("Account ID: %v Name: %v Email: %v\n", aws.ToString(y.AccountId), aws.ToString(y.AccountName), aws.ToString(y.EmailAddress))
		}
	}
}

var profile string

type promptContent struct {
	errorMsg string
	label    string
}

func ssm_tunnel(bastionHostID string, rdsURL string, localPort int) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	in := ssmclient.PortForwardingInput{
		Target:     bastionHostID,
		RemotePort: 3306, // constant
		LocalPort:  localPort,
		Host:       rdsURL,
	}

	// Alternatively, can be called as ssmclient.PortluginSession(cfg, tgt) to use the AWS-managed SSM session client code
	//log.Fatal(ssmclient.PortForwardingSession(cfg, &in))
	log.Fatal(ssmclient.PortPluginSession(cfg, &in))
}

func promptGetInput(pc promptContent) string {

	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(pc.errorMsg)
		}
		// Check if input is a valid port number
		port, err := strconv.Atoi(input)
		if err != nil {
			return errors.New("invalid port number") // Fix: Changed error string to lowercase
		}
		if port < 22 || port > 65535 {
			return errors.New("invalid port number") // Fix: Changed error string to lowercase
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     pc.label,
		Default:   "3306",
		Templates: templates,
		Validate:  validate,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return result
}

var DisableAccountAliasEnvVarName = "AWS_WHOAMI_DISABLE_ACCOUNT_ALIAS"

type Whoami struct {
	Account          string
	AccountAliases   []string
	Arn              string
	Type             string
	Name             string
	RoleSessionName  *string
	UserId           string
	Region           string
	SSOPermissionSet *string
}

type WhoamiParams struct {
	DisableAccountAlias       bool
	DisableAccountAliasValues []string
}

func NewWhoamiParams() WhoamiParams {
	var params WhoamiParams
	disableAccountAliasValue := os.Getenv(DisableAccountAliasEnvVarName)
	populateDisableAccountAlias(&params, disableAccountAliasValue)
	return params
}

func populateDisableAccountAlias(params *WhoamiParams, disableAccountAliasValue string) {
	switch strings.ToLower(disableAccountAliasValue) {
	case "":
		fallthrough
	case "0":
		fallthrough
	case "false":
		params.DisableAccountAlias = false
		params.DisableAccountAliasValues = nil
		return
	case "1":
		fallthrough
	case "true":
		params.DisableAccountAlias = true
		params.DisableAccountAliasValues = nil
		return
	default:
		accounts := strings.Split(disableAccountAliasValue, ",")
		if len(accounts) > 0 {
			params.DisableAccountAlias = true
			params.DisableAccountAliasValues = accounts
			return
		} else {
			params.DisableAccountAlias = false
			params.DisableAccountAliasValues = nil
			return
		}
	}
}

func (params WhoamiParams) GetDisableAccountAlias(whoami Whoami) bool {
	if !params.DisableAccountAlias {
		return false
	}
	if params.DisableAccountAliasValues == nil {
		return true
	}
	for _, disabledValue := range params.DisableAccountAliasValues {
		if strings.HasPrefix(whoami.Account, disabledValue) || strings.HasSuffix(whoami.Account, disabledValue) {
			return true
		}
		if whoami.Arn == disabledValue || whoami.Name == disabledValue {
			return true
		}
		if whoami.RoleSessionName != nil && *whoami.RoleSessionName == disabledValue {
			return true
		}
		if whoami.SSOPermissionSet != nil && *whoami.SSOPermissionSet == disabledValue {
			return true
		}
	}
	return false
}

func populateWhoamiFromGetCallerIdentityOutput(whoami *Whoami, getCallerIdentityOutput sts.GetCallerIdentityOutput) error {
	whoami.Account = *getCallerIdentityOutput.Account
	whoami.Arn = *getCallerIdentityOutput.Arn
	whoami.UserId = *getCallerIdentityOutput.UserId

	arnFields := strings.Split(whoami.Arn, ":")

	var arnResourceFields []string
	if arnFields[len(arnFields)-1] == "root" {
		arnResourceFields = []string{"root", "root"}
	} else {
		arnResourceFields = strings.SplitN(arnFields[len(arnFields)-1], "/", 2)
		if len(arnResourceFields) < 2 {
			return fmt.Errorf("arn %v has an unknown format", whoami.Arn)
		}
	}

	whoami.Type = arnResourceFields[0]
	if whoami.Type == "assumed-role" {
		nameFields := strings.SplitN(arnResourceFields[1], "/", 2)
		if len(arnResourceFields) < 2 {
			return fmt.Errorf("arn %v has an unknown format", whoami.Arn)
		}
		whoami.Name = nameFields[0]
		whoami.RoleSessionName = &nameFields[1]
	} else if whoami.Type == "user" {
		nameFields := strings.Split(arnResourceFields[1], "/")
		whoami.Name = nameFields[len(nameFields)-1]
	} else {
		whoami.Name = arnResourceFields[1]
	}

	if whoami.Type == "assumed-role" && strings.HasPrefix(whoami.Name, "AWSReservedSSO") {
		nameFields := strings.Split(whoami.Name, "_")
		if len(nameFields) >= 3 {
			permSetStr := strings.Join(nameFields[1:len(nameFields)-1], "_")
			whoami.SSOPermissionSet = &permSetStr
		}
	}

	return nil
}

func NewWhoami(awsConfig aws.Config, params WhoamiParams) (Whoami, error) {
	stsClient := sts.NewFromConfig(awsConfig)

	getCallerIdentityOutput, err := stsClient.GetCallerIdentity(context.TODO(), nil)

	if err != nil {
		return Whoami{}, err
	}

	var whoami Whoami
	whoami.AccountAliases = make([]string, 0, 1)

	whoami.Region = awsConfig.Region

	err = populateWhoamiFromGetCallerIdentityOutput(&whoami, *getCallerIdentityOutput)

	if err != nil {
		return whoami, err
	}

	if !params.GetDisableAccountAlias(whoami) {
		iam_client := iam.NewFromConfig(awsConfig)

		// pedantry
		paginator := iam.NewListAccountAliasesPaginator(iam_client, nil)

		for paginator.HasMorePages() {
			output, err := paginator.NextPage(context.TODO())
			if err != nil {
				var apiErr smithy.APIError
				if errors.As(err, &apiErr) && apiErr.ErrorCode() == "AccessDenied" {
					break
				} else {
					return whoami, err
				}
			}
			whoami.AccountAliases = append(whoami.AccountAliases, output.AccountAliases...)
		}
	}

	return whoami, nil
}

type record struct {
	field string
	value string
}

func getTypeNameRecord(whoami Whoami) record {
	if whoami.Type == "root" {
		return record{"Type: ", "root"}
	}
	fields := strings.Split(whoami.Type, "-")
	typeParts := make([]string, 0, 3)
	for _, field := range fields {
		s := strings.ToUpper(field[:1]) + field[1:] // ok because always ASCII
		typeParts = append(typeParts, s)
	}
	typeParts = append(typeParts, ": ")
	return record{strings.Join(typeParts, ""), whoami.Name}
}

func (whoami Whoami) Format() string {
	records := make([]record, 0, 7)
	records = append(records, record{"Account: ", whoami.Account})
	for _, alias := range whoami.AccountAliases {
		records = append(records, record{"", alias})
	}
	records = append(records, record{"Region: ", whoami.Region})
	if whoami.SSOPermissionSet != nil {
		records = append(records, record{"AWS SSO: ", *whoami.SSOPermissionSet})
	} else {
		records = append(records, getTypeNameRecord(whoami))
	}
	if whoami.RoleSessionName != nil {
		records = append(records, record{"RoleSessionName: ", *whoami.RoleSessionName})
	}
	records = append(records, record{"UserId: ", whoami.UserId})
	records = append(records, record{"Arn: ", whoami.Arn})

	var maxLen int = 0
	for _, rec := range records {
		if len(rec.field) > maxLen {
			maxLen = len(rec.field)
		}
	}

	lines := make([]string, 0, 7)
	for _, rec := range records {
		lines = append(lines, rec.field+strings.Repeat(" ", maxLen-len(rec.field))+rec.value)
	}

	return strings.Join(lines, "\n")
}

func showIamDetails() {
	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	whoamiParams := NewWhoamiParams()
	Whoami, err := NewWhoami(awsConfig, whoamiParams)

	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	fmt.Println(Whoami.Format())
}

func loadAllAWSProfiles() ([]string, error) {
	configFile := os.Getenv("HOME") + "/.aws/config"
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var profiles []string
	scanner := bufio.NewScanner(file)
	currentProfile := ""
	shouldAddProfile := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[profile ") {
			if shouldAddProfile {
				profiles = append(profiles, currentProfile)
			}
			currentProfile = line[9 : len(line)-1]
			shouldAddProfile = false // Reset for the new profile
		}

		if true { // {strings.Contains(line, "sso_session = ito") {
			shouldAddProfile = true
		}
	}

	// Check the last profile processed by the loop
	profiles = append(profiles, currentProfile)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}
