package control

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	brokerMod "github.com/fornellas/cgs/broker"
	grblMod "github.com/fornellas/cgs/grbl"
)

// TrackedState is the virtual Grbl state tracked by StateTracker.
type TrackedState struct {
	State    grblMod.State
	SubState *string
	Error    error
}

var UnknownTrackedState = &TrackedState{
	State: grblMod.StateUnknown,
}

// StateTracker keeps track of Grbl state, handling corner cases where
// [grblMod.StatusReportPushMessage] isn't enough / does not work (ie: $H, Alarm push message).
type StateTracker struct {
	*brokerMod.Broker[*TrackedState]

	mu sync.Mutex

	homeOverride     bool
	machineState     *grblMod.MachineState
	alarmPushMessage *grblMod.AlarmPushMessage

	lastPublishedTrackedState *TrackedState
}

func NewStateTracker() *StateTracker {
	return &StateTracker{
		Broker: brokerMod.NewBroker[*TrackedState](),
	}
}

func (st *StateTracker) getTrackedState() *TrackedState {
	if st.homeOverride {
		return &TrackedState{
			State: grblMod.StateHome,
		}
	}

	if st.alarmPushMessage != nil {
		return &TrackedState{
			State: grblMod.StateAlarm,
			Error: st.alarmPushMessage.Error(),
		}
	}

	if st.machineState != nil {
		var subState *string
		if subStateString := st.machineState.SubStateString(); subStateString != "" {
			subState = &subStateString
		}
		return &TrackedState{
			State:    st.machineState.State,
			SubState: subState,
		}
	}

	return &TrackedState{
		State: grblMod.StateUnknown,
	}
}

func (st *StateTracker) publish() {
	trackedState := st.getTrackedState()
	if reflect.DeepEqual(st.lastPublishedTrackedState, trackedState) {
		return
	}

	st.Broker.Publish(trackedState)

	st.lastPublishedTrackedState = trackedState
}

// Grbl stops responding to status report queries while homing. This enable overriding the Home
// state while home is ongoing.
func (st *StateTracker) HomeOverride(homeOverride bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.homeOverride = homeOverride
	st.publish()
}

// Worker processes push messages from Grbl and updates the tracked state accordingly.
// The worker runs until ctx is canceled or the push message channel is closed.
// It publishes state changes via the Broker and returns any error encountered.
func (st *StateTracker) Worker(ctx context.Context, pushMessageCh <-chan grblMod.PushMessage) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			st.Broker.Close()
			return err
		case pushMessage, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}

			st.mu.Lock()

			if _, ok := pushMessage.(*grblMod.WelcomePushMessage); ok {
				st.homeOverride = false
				st.machineState = nil
				st.alarmPushMessage = nil
				st.lastPublishedTrackedState = nil
			}
			if alarmPushMessage, ok := pushMessage.(*grblMod.AlarmPushMessage); ok {
				st.alarmPushMessage = alarmPushMessage
			}
			if statusReportPushMessage, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
				st.machineState = &statusReportPushMessage.MachineState
			}

			st.publish()

			st.mu.Unlock()
		}
	}
}
