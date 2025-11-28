package control

import (
	"context"

	grblMod "github.com/fornellas/cgs/grbl"
)

// PushMessageProcessor interface for receiving and processing [grblMod.PushMessage].
type PushMessageProcessor interface {
	ProcessPushMessage(context.Context, grblMod.PushMessage)
}
