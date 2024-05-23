package ssmclient

import (
	"context"
	logger "log"
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

func PluginSession(cfg aws.Config, input *ssm.StartSessionInput) error {
	out, err := ssm.NewFromConfig(cfg).StartSession(context.Background(), input)
	if err != nil {
		return err
	}

	ep, err := ssm.NewDefaultEndpointResolver().ResolveEndpoint(cfg.Region, ssm.EndpointResolverOptions{})
	if err != nil {
		return err
	}

	// use a signal handler vs. defer since defer operates after an escape from the outer loop
	// and we can't trust the data channel connection state at that point.  Intercepting signals
	// means we're probably trying to shutdown somewhere in the outer loop, and there's a good
	// possibility that the data channel is still valid
	installSignalHandler(datachannel.DataChannel{})

	ssmSession := new(session.Session)
	ssmSession.SessionId = *out.SessionId
	ssmSession.StreamUrl = *out.StreamUrl
	ssmSession.TokenValue = *out.TokenValue
	ssmSession.Endpoint = ep.URL
	ssmSession.ClientId = uuid.NewString()
	ssmSession.TargetId = *input.Target
	ssmSession.DataChannel = &datachannel.DataChannel{}

	return ssmSession.Execute(log.Logger(false, ssmSession.ClientId))
}

// shared with ssh.go.
func installSignalHandler(c datachannel.DataChannel) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Printf("Got signal: %s, shutting down", sig.String())

		//_ = c..TerminateSession()
		_ = c.Close(log.Logger(false, ""))

		os.Exit(0)
	}()
}
