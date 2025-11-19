package control

import (
	"context"
	"errors"
	"fmt"

	grblMod "github.com/fornellas/cgs/grbl"
)

type MessageProcessor struct {
	pushMessageCh    chan grblMod.Message
	appManager       *AppManager
	controlPrimitive *ControlPrimitive
}

func NewMessageProcessor(
	pushMessageCh chan grblMod.Message,
	appManager *AppManager,
	controlPrimitive *ControlPrimitive,
) *MessageProcessor {
	return &MessageProcessor{
		pushMessageCh:    pushMessageCh,
		appManager:       appManager,
		controlPrimitive: controlPrimitive,
	}
}

func (mp *MessageProcessor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case message, ok := <-mp.pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}

			mp.appManager.ProcessMessage(message)
			mp.controlPrimitive.ProcessMessage(message)
		}
	}
}
