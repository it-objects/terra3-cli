package ssmclient

import (
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// PortForwardingInput configures the port forwarding session parameters.
// Target is the EC2 instance ID to establish the session with.
// RemotePort is the port on the EC2 instance to connect to.
// LocalPort is the port on the local host to listen to.  If not provided, a random port will be used.
type PortForwardingInput struct {
	Target     string
	RemotePort int
	LocalPort  int
	Host       string
}

func PortPluginSession(cfg aws.Config, opts *PortForwardingInput) error {
	in := &ssm.StartSessionInput{
		DocumentName: aws.String("AWS-StartPortForwardingSessionToRemoteHost"),
		Target:       aws.String(opts.Target),
		Parameters: map[string][]string{
			"localPortNumber": {strconv.Itoa(opts.LocalPort)},
			"portNumber":      {strconv.Itoa(opts.RemotePort)},
			"host":            {opts.Host},
		},
	}

	return PluginSession(cfg, in)
}
