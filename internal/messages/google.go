package messages

import (
	"auto-myself-api/internal/google"
	"context"
	"time"

	"cloud.google.com/go/pubsub/v2"
)

type UpdateHandler func(ctx context.Context, msg UpdateMessage) error

type UpdateMessage struct {
	ID          string
	Data        []byte
	Attributes  map[string]string
	PublishedAt time.Time
}

func SubscribeUpdatesChannel(ctx context.Context, subscriptionName string, handler UpdateHandler) error {
	projectID := google.GetNumericProjectID()
	clientSub, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer clientSub.Close()

	sub := clientSub.Subscriber(subscriptionName)

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {

		err := handler(ctx, UpdateMessage{
			ID:          msg.ID,
			Data:        msg.Data,
			Attributes:  msg.Attributes,
			PublishedAt: msg.PublishTime,
		})

		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})

	if err != nil {
		return err
	}

	return nil
}
