package docker_utils

import (
	"testing"
)

func TestParseAndHandleEvent(t *testing.T) {
	evtJSON := []byte(`{"Action":"die","Actor":{"Attributes":{"name":"test-container"}}}`)
	var capturedName, capturedAction string

	parseAndHandleEvent(evtJSON, func(actorName, action string) {
		capturedName = actorName
		capturedAction = action
	})

	if capturedName != "test-container" || capturedAction != "die" {
		t.Errorf("Unexpected event handling: name='%s', action='%s'", capturedName, capturedAction)
	}
}

func TestParseAndHandleEventIgnoredAction(t *testing.T) {
	evtJSON := []byte(`{"Action":"start","Actor":{"Attributes":{"name":"test-container"}}}`)
	called := false

	parseAndHandleEvent(evtJSON, func(actorName, action string) {
		called = true
	})

	if called {
		t.Error("Expected 'start' action to be ignored by container death handler")
	}
}
