package ssmclient

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/session-manager-plugin/src/datachannel"
	"github.com/aws/session-manager-plugin/src/log"
	"github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session"
	_ "github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session/portsession"
	"github.com/google/uuid"
)

// PluginSession starts an SSM session with AWS credentials and runs the data channel.
// Used by the legacy AWS-profile-based port-forward path.
func PluginSession(cfg aws.Config, input *ssm.StartSessionInput) error {
	out, err := ssm.NewFromConfig(cfg).StartSession(context.Background(), input)
	if err != nil {
		return err
	}

	installExitOnSignal()
	return RunSession(cfg.Region, *out.SessionId, *out.StreamUrl, *out.TokenValue, aws.ToString(input.Target))
}

// RunSession connects the Session Manager data channel using only the opaque
// session credentials returned by StartSession. No AWS credentials are required.
// Returns when the data channel closes (idle timeout, remote end, or error).
func RunSession(region, sessionId, streamUrl, tokenValue, targetId string) error {
	ep, err := ssm.NewDefaultEndpointResolver().ResolveEndpoint(region, ssm.EndpointResolverOptions{})
	if err != nil {
		return fmt.Errorf("resolve SSM endpoint: %w", err)
	}

	ssmSession := new(session.Session)
	ssmSession.SessionId = sessionId
	ssmSession.StreamUrl = streamUrl
	ssmSession.TokenValue = tokenValue
	ssmSession.Endpoint = ep.URL
	ssmSession.ClientId = uuid.NewString()
	ssmSession.TargetId = targetId
	ssmSession.DataChannel = &datachannel.DataChannel{}

	return ssmSession.Execute(log.Logger(false, ssmSession.ClientId))
}

func installExitOnSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("Got signal: %s, shutting down...\n", sig.String())
		os.Exit(0)
	}()
}
