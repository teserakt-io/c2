package protocols

import (
	"context"
	"encoding/base64"
	"fmt"

	log "github.com/sirupsen/logrus"
	cloudiot "google.golang.org/api/cloudiot/v1"

	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/models"
)

type gcpClient struct {
	projectID        string
	region           string
	registryID       string
	commandSubFolder string

	logger log.FieldLogger
}

var _ PubSubClient = (*gcpClient)(nil)

// NewGCPClient returns a new PubSubClient able to interact with GCP IoT Core apis
// Currently only support authorizing the client using Application Default Credentials.
// See https://g.co/dv/identity/protocols/application-default-credentials
// TLDR: run `gcloud auth login` on the host running the C2, then start the C2.
//
// TODO: better auth
//
// Only the Publish method is implemented, to send commands to devices.
// Other features, such as topic subscription and un-subscription,
// are not available due to GCP restrictions, and are just no-op.
func NewGCPClient(cfg config.GCPCfg, logger log.FieldLogger) PubSubClient {
	return &gcpClient{
		projectID:        cfg.ProjectID,
		region:           cfg.Region,
		registryID:       cfg.RegistryID,
		commandSubFolder: cfg.CommandSubFolder,

		logger: logger,
	}
}

func (c *gcpClient) Connect() error {
	return nil
}

func (c *gcpClient) Disconnect() error {
	return nil
}

func (c *gcpClient) SubscribeToTopics(ctx context.Context, topics []string) error {
	return nil
}

func (c *gcpClient) SubscribeToTopic(ctx context.Context, topic string) error {
	return nil
}

func (c *gcpClient) UnsubscribeFromTopic(ctx context.Context, topic string) error {
	return nil
}

func (c *gcpClient) Publish(ctx context.Context, payload []byte, client models.Client, qos byte) error {
	iotClient, err := cloudiot.NewService(ctx)
	if err != nil {
		return err
	}

	name := fmt.Sprintf(
		"projects/%s/locations/%s/registries/%s/devices/%s",
		c.projectID,
		c.region,
		c.registryID,
		client.Name,
	)
	req := cloudiot.SendCommandToDeviceRequest{
		BinaryData: base64.StdEncoding.EncodeToString(payload),
		Subfolder:  c.commandSubFolder,
	}

	_, err = iotClient.Projects.Locations.Registries.Devices.SendCommandToDevice(name, &req).Do()
	if err != nil {
		return err
	}

	return nil
}

func (c *gcpClient) ValidateTopic(topic string) error {
	return nil
}
