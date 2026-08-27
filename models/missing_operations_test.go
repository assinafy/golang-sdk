package models

import (
	"encoding/json"
	"testing"
)

func TestCreateAccountRequestMarshal(t *testing.T) {
	sender := NotificationSenderAccount
	got, err := json.Marshal(CreateAccountRequest{Name: "Acme", NotificationSenderType: &sender})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `{"name":"Acme","notification_sender_type":"Account"}` {
		t.Errorf("body = %s", got)
	}
}

func TestDocumentStatsRowUnmarshal(t *testing.T) {
	var row DocumentStatsRow
	if err := json.Unmarshal([]byte(`{"period":"2026-06","documents_uploaded":2,"documents_sent":1,"signature_requests":3,"signature_requests_notification_email":2,"signature_requests_notification_whatsapp":1,"signature_requests_notification_bypass":1,"signature_requests_verification_email":1,"signature_requests_verification_whatsapp":1,"signature_requests_verification_bypass":0,"signature_requests_verification_digital_certificate":1,"signature_requests_viewed":2,"signature_requests_completed":1,"documents_certified":1}`), &row); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if row.Period != "2026-06" || row.DocumentsUploaded != 2 ||
		row.SignatureRequestsNotificationEmail != 2 || row.SignatureRequestsNotificationWhatsApp != 1 ||
		row.SignatureRequestsNotificationBypass != 1 || row.SignatureRequestsVerificationEmail != 1 ||
		row.SignatureRequestsVerificationWhatsApp != 1 || row.SignatureRequestsVerificationBypass != 0 ||
		row.SignatureRequestsVerificationDigitalCertificate != 1 || row.DocumentsCertified != 1 {
		t.Errorf("row = %+v", row)
	}
}

func TestNotificationPreferencesJSON(t *testing.T) {
	prefs := NotificationPreferences{
		NotificationPreferenceDocumentCompleted: true,
		NotificationPreferenceSignerDeclined:    false,
	}
	body, err := json.Marshal(prefs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var roundTrip NotificationPreferences
	if err := json.Unmarshal(body, &roundTrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(roundTrip) != 2 || !roundTrip[NotificationPreferenceDocumentCompleted] || roundTrip[NotificationPreferenceSignerDeclined] {
		t.Errorf("preferences = %v", roundTrip)
	}
}
