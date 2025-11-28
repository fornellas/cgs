package control

import (
	"context"
	"reflect"
	"sync"

	grblMod "github.com/fornellas/cgs/grbl"
)

// TrackedState is the virtual Grbl state tracked by StateTrackerPushMessageProcessor.
type TrackedState struct {
	State    grblMod.State
	SubState *string
	Error    error
}

// StateTracker implements [PushMessageProcessor] and keeps track of Grbl state,
// handling corner cases where [grblMod.StatusReportPushMessage] isn't enough / does not work.
type StateTracker struct {
	mu sync.Mutex

	homeOverride     bool
	machineState     *grblMod.MachineState
	alarmPushMessage *grblMod.AlarmPushMessage

	lastTrackedState *TrackedState

	subscribers map[string]chan *TrackedState
}

func (s *StateTracker) getTrackedState() *TrackedState {
	if s.alarmPushMessage != nil {
		return &TrackedState{
			State: grblMod.StateAlarm,
			Error: s.alarmPushMessage.Error(),
		}
	}

	if s.machineState != nil {
		subState := s.machineState.SubStateString()
		return &TrackedState{
			State:    s.machineState.State,
			SubState: &subState,
		}
	}

	return &TrackedState{
		State: grblMod.StateUnknown,
	}
}

func (s *StateTracker) publish() {
	trackedState := s.getTrackedState()
	if reflect.DeepEqual(s.lastTrackedState, trackedState) {
		return
	}

	for _, trackedStateCh := range s.subscribers {
		go func() { trackedStateCh <- trackedState }()
	}

	s.lastTrackedState = trackedState
}

// Grbl stops responding to status report queries while homing. This enable overriding the Home
// state while home is ongoing.
func (s *StateTracker) HomeOverride(homeOverride bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.homeOverride = homeOverride
	s.publish()
}

func (s *StateTracker) ProcessPushMessage(
	ctx context.Context, pushMessage grblMod.PushMessage,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := pushMessage.(*grblMod.WelcomePushMessage); ok {
		s.homeOverride = false
		s.machineState = nil
		s.alarmPushMessage = nil
		s.lastTrackedState = nil
	}
	if alarmPushMessage, ok := pushMessage.(*grblMod.AlarmPushMessage); ok {
		s.alarmPushMessage = alarmPushMessage
	}
	if statusReportPushMessage, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
		s.machineState = &statusReportPushMessage.MachineState
	}
	s.publish()
}

// Adds a new subscriber to state changes
func (s *StateTracker) Subscribe(name string) <-chan *TrackedState {
	s.mu.Lock()
	defer s.mu.Unlock()

	trackedStateCh := make(chan *TrackedState, 10)

	s.subscribers[name] = trackedStateCh

	return trackedStateCh
}

func (s *StateTracker) Close() {
	for _, trackedStateCh := range s.subscribers {
		close(trackedStateCh)
	}
}
