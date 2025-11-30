package control

import (
	"context"
	"fmt"

	brokerMod "github.com/fornellas/cgs/broker"
	grblMod "github.com/fornellas/cgs/grbl"
)

type PushMessageBroker struct {
	*brokerMod.Broker[grblMod.PushMessage]
}

func NewPushMessageBroker() *PushMessageBroker {
	return &PushMessageBroker{
		Broker: brokerMod.NewBroker[grblMod.PushMessage](),
	}
}

func (s *PushMessageBroker) Worker(ctx context.Context, pushMessageCh <-chan grblMod.PushMessage) error {
	for {
		select {
		case <-ctx.Done():
			s.Broker.Close()
			return ctx.Err()
		case pushMessage, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}
			s.Broker.Publish(pushMessage)
		}
	}
}
