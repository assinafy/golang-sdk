package models

import (
	"encoding/json"
	"testing"
)

func TestScheduledSendCompletionPayload(t *testing.T) {
	var send ScheduledSend
	if err := json.Unmarshal([]byte(`{"id":"send","type":"envelope","document_ids":["doc"],"status":"sent","send_at":"2026-12-01T12:00:00Z","assignment_id":null,"envelope_id":"envelope","failure_reason":null,"created_at":"2026-11-24T12:00:00Z","processed_at":"2026-12-01T12:00:01Z"}`), &send); err != nil {
		t.Fatal(err)
	}
	if send.EnvelopeID == nil || *send.EnvelopeID != "envelope" || send.ProcessedAt == nil || send.Status != "sent" {
		t.Fatalf("send = %+v", send)
	}
	encoded, err := json.Marshal(send)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"assignment_id", "failure_reason"} {
		if value, exists := fields[name]; !exists || value != nil {
			t.Errorf("%s = %v, present %v", name, value, exists)
		}
	}
}

func TestEstimateSignerCapacity(t *testing.T) {
	encoded, err := json.Marshal(EstimateAssignmentCostRequest{Signers: []EstimateAssignmentCostSigner{{SignAs: SignerCapacityWitness}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"signers":[{"sign_as":"Witness"}]}` {
		t.Fatalf("estimate = %s", encoded)
	}
}
