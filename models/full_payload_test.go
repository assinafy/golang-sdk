package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFullResponsePayloads(t *testing.T) {
	t.Run("document", func(t *testing.T) {
		const fixture = `{
			"resource":"document","id":"doc_1","account_id":"account_1","template_id":"template_1","name":"Agreement.pdf","status":"certificated",
			"artifacts":{"original":"https://example.test/documents/doc_1/original","certificated":"https://example.test/documents/doc_1/certificated","certificate-page":"https://example.test/documents/doc_1/certificate-page","pades":"https://example.test/documents/doc_1/pades","bundle":"https://example.test/documents/doc_1/bundle","thumbnail":"https://example.test/documents/doc_1/thumbnail"},
			"is_closed":true,"signing_url":"https://example.test/sign/doc_1","decline_reason":"duplicate",
			"declined_by":{"resource":"signer","id":"signer_1","full_name":"Example Signer","email":"signer@example.test","has_accepted_terms":true},
			"tags":[{"resource":"tag","id":"tag_1","name":"Legal","color":"112233","created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z"}],
			"created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z",
			"assignment":{"resource":"assignment","id":"assignment_1","document_id":"doc_1","method":"virtual","status":"completed"},
			"pages":[{"id":"page_1","number":1,"height":1754,"width":1240,"download_url":"https://example.test/pages/page_1"}],
			"activities":[{"id":7,"event":"document.certificated","message":"Document certificated","payload":{"document_id":"doc_1"},"origin":{"ip":"192.0.2.1","user-agent":"Assinafy-Test/1.0"},"created_at":"2026-01-01T11:00:00Z"}],
			"current_signer":{"resource":"signer","id":"signer_2","full_name":"Current Signer","email":"current@example.test","has_accepted_terms":false},
			"download_url":"https://example.test/documents/doc_1/download","download_final_url":"https://example.test/documents/doc_1/final"
		}`
		document := assertPayloadRoundTrip[Document](t, fixture)
		if document.TemplateID == nil || *document.TemplateID != "template_1" || document.Artifacts == nil ||
			document.Assignment == nil || document.Assignment.ID != "assignment_1" || len(document.Pages) != 1 ||
			document.CurrentSigner == nil || document.CurrentSigner.ID != "signer_2" || document.DownloadURL == nil ||
			*document.DownloadURL == "" || document.DownloadFinalURL == nil || *document.DownloadFinalURL == "" {
			t.Fatalf("document relationship or URL fields were not decoded: %+v", document)
		}
		for name, value := range map[string]*string{
			"original": document.Artifacts.Original, "certificated": document.Artifacts.Certificated,
			"certificate-page": document.Artifacts.CertificatePage, "pades": document.Artifacts.PAdES,
			"bundle": document.Artifacts.Bundle, "thumbnail": document.Artifacts.Thumbnail,
		} {
			if value == nil || *value == "" {
				t.Errorf("artifact %q was not decoded", name)
			}
		}
	})

	t.Run("assignment", func(t *testing.T) {
		const fixture = `{
			"resource":"assignment","id":"assignment_1","document_id":"doc_1","sender_email":"sender@example.test","method":"collect","status":"completed","expiration":"2026-02-01","expires_at":"2026-02-01T10:00:00Z","message":"Please sign",
			"signers":[{"resource":"signer","id":"signer_1","full_name":"Example Signer","email":"signer@example.test","whatsapp_phone_number":"+15555550100","has_accepted_terms":true,"has_signature":true,"has_initial":true,"is_signature_reusable":true,"verification_method":"Email","notification_methods":["Email"],"completed":true,"step":1,"notified":true,"notification_history":[{"event":"signature_request","status":"sent","sent_at":"2026-01-01T10:01:00Z"}]}],
			"copy_receivers":[{"id":"copy_1","full_name":"Copy Receiver","email":"copy@example.test","has_accepted_terms":false}],
			"items":[{"id":"item_1","page":{"id":"page_1","number":1,"height":1754,"width":1240,"download_url":"https://example.test/pages/page_1"},"signer":{"id":"signer_1","full_name":"Example Signer","has_accepted_terms":true},"field":{"resource":"field","id":"field_1","name":"Signature","type":"signature","is_pre_defined":true,"is_active":true,"is_required":true,"is_standard":true,"is_read_only":false,"is_visible":true},"display_settings":{"left":10,"top":20,"width":200,"height":50,"fontSize":12},"value":"signed","completed":true}],
			"summary":{"signer_count":1,"completed_count":1,"signers":[{"id":"signer_1","full_name":"Example Signer","has_accepted_terms":true,"completed":true}]},
			"signing_urls":[{"signer_id":"signer_1","url":"https://example.test/sign/signer_1"}],"completed_at":"2026-01-01T11:00:00Z","created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z"
		}`
		assignment := assertPayloadRoundTrip[Assignment](t, fixture)
		if assignment.Method != MethodCollect || len(assignment.Signers) != 1 || len(assignment.CopyReceivers) != 1 ||
			len(assignment.Items) != 1 || assignment.Items[0].Page == nil || assignment.Items[0].Signer == nil ||
			assignment.Items[0].Field == nil || assignment.Summary == nil || len(assignment.SigningURLs) != 1 ||
			assignment.CompletedAt == nil || assignment.CompletedAt.String() != "2026-01-01T11:00:00Z" {
			t.Fatalf("assignment fields were not decoded: %+v", assignment)
		}
	})

	t.Run("template", func(t *testing.T) {
		const fixture = `{
			"resource":"template","id":"template_1","name":"Employment Agreement","document_name":"Agreement.pdf","message":"Please review","status":"ready",
			"pages":[{"id":"page_1","number":1,"height":1754,"width":1240,"download_url":"https://example.test/template-pages/page_1","fields":[{"id":"placement_1","field_id":"field_1","role_id":"role_1","label":"Signature","display_settings":{"left":10,"top":20,"width":200,"height":50},"created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z"}]}],
			"roles":[{"id":"role_1","name":"Employee","description":"Employee signer","assignment_type":"signature","created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z"}],
			"tags":[{"id":"tag_1","name":"HR"}],"default_document_tags":[{"id":"tag_2","name":"Generated"}],"created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T11:00:00Z"
		}`
		template := assertPayloadRoundTrip[Template](t, fixture)
		if template.DocumentName == nil || template.Message == nil || template.Status != TemplateStatusReady ||
			len(template.Pages) != 1 || len(template.Pages[0].Fields) != 1 || len(template.Roles) != 1 ||
			len(template.Tags) != 1 || len(template.DefaultDocumentTags) != 1 {
			t.Fatalf("template fields were not decoded: %+v", template)
		}
	})

	t.Run("webhook dispatch", func(t *testing.T) {
		const fixture = `{
			"resource":"activity_dispatching_history","id":"dispatch_1","event":"document_ready","activity_id":42,"endpoint":"https://hooks.example.test/assinafy",
			"payload":{"document_id":"doc_1","attempt":2},"delivered":false,"http_status":500,"response_body":"temporary failure","error":"subscriber returned an error",
			"created_at":"2026-01-01T10:00:00Z","updated_at":"2026-01-01T10:01:00Z"
		}`
		dispatch := assertPayloadRoundTrip[WebhookDispatch](t, fixture)
		if dispatch.ActivityID != 42 || dispatch.Endpoint == nil || dispatch.HTTPStatus == nil ||
			*dispatch.HTTPStatus != 500 || dispatch.ResponseBody == nil || dispatch.Error == nil ||
			dispatch.Payload["attempt"] != float64(2) {
			t.Fatalf("webhook dispatch fields were not decoded: %+v", dispatch)
		}
	})

	t.Run("webhook payload", func(t *testing.T) {
		const fixture = `{
			"id":42,"event":"document_ready","message":"Document certificated","payload":{"document_id":"doc_1","page_count":2},
			"origin":{"ip":"192.0.2.1","user-agent":"Assinafy-Test/1.0"},"created_at":1705316400,
			"subject":{"type":"Account","id":"account_1","version":2},"object":{"type":"Document","id":"doc_1","status":"certificated","page_count":2},"account_id":"account_1"
		}`
		payload := assertPayloadRoundTrip[WebhookPayload](t, fixture)
		if payload.CreatedAt.String() != "1705316400" || payload.Subject["version"] != float64(2) ||
			payload.Object["page_count"] != float64(2) || payload.Origin == nil || payload.AccountID != "account_1" {
			t.Fatalf("webhook payload fields were not decoded: %+v", payload)
		}
	})
}

func assertPayloadRoundTrip[T any](t *testing.T, fixture string) T {
	t.Helper()
	var value T
	if err := json.Unmarshal([]byte(fixture), &value); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal decoded payload: %v", err)
	}
	var expected, actual map[string]any
	if err := json.Unmarshal([]byte(fixture), &expected); err != nil {
		t.Fatalf("unmarshal expected payload: %v", err)
	}
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatalf("unmarshal round-trip payload: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("payload fields changed after decode:\n got: %#v\nwant: %#v", actual, expected)
	}
	return value
}
