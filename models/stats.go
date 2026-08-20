package models

// StatsGranularity controls the period represented by each document statistics row.
type StatsGranularity string

// Documented values for StatsGranularity.
const (
	// StatsGranularityMonthly returns one row per calendar month.
	StatsGranularityMonthly StatsGranularity = "monthly"
	// StatsGranularityDaily returns one row per day of StatsParams.Month.
	StatsGranularityDaily StatsGranularity = "daily"
)

// StatsParams are the query parameters accepted by account and user statistics endpoints.
type StatsParams struct {
	// Granularity defaults to StatsGranularityMonthly when omitted.
	Granularity StatsGranularity
	// Month selects a YYYY-MM month and is required for daily granularity.
	Month string
}

// DocumentStatsRow contains one period of document-funnel KPIs.
type DocumentStatsRow struct {
	// Period is YYYY-MM for monthly rows or YYYY-MM-DD for daily rows.
	Period string `json:"period"`
	// DocumentsUploaded is the number of documents uploaded in Period.
	DocumentsUploaded int `json:"documents_uploaded"`
	// DocumentsSent is the number of documents sent for signature in Period.
	DocumentsSent int `json:"documents_sent"`
	// SignatureRequests is the total number of signer invitations in Period.
	SignatureRequests int `json:"signature_requests"`
	// SignatureRequestsEmail is the number of invitations sent by email.
	SignatureRequestsEmail int `json:"signature_requests_email"`
	// SignatureRequestsWhatsApp is the number sent through WhatsApp.
	SignatureRequestsWhatsApp int `json:"signature_requests_whatsapp"`
	// SignatureRequestsViewed is the number of invitations viewed by signers.
	SignatureRequestsViewed int `json:"signature_requests_viewed"`
	// SignatureRequestsCompleted is the number completed by signers.
	SignatureRequestsCompleted int `json:"signature_requests_completed"`
	// DocumentsCertified is the number of documents certified in Period.
	DocumentsCertified int `json:"documents_certified"`
}

// NotificationPreference identifies one configurable owner-facing email notification.
type NotificationPreference string

// Documented notification preference codes.
const (
	// NotificationPreferenceDocumentCompleted controls document-completion email.
	NotificationPreferenceDocumentCompleted NotificationPreference = "DocumentCompleted"
	// NotificationPreferenceSignerDeclined controls signer-declined email.
	NotificationPreferenceSignerDeclined NotificationPreference = "SignerDeclined"
	// NotificationPreferenceDocumentCancelled controls document-cancelled email.
	NotificationPreferenceDocumentCancelled NotificationPreference = "DocumentCancelled"
	// NotificationPreferenceDocumentAboutToExpire controls pre-expiration email.
	NotificationPreferenceDocumentAboutToExpire NotificationPreference = "DocumentAboutToExpire"
	// NotificationPreferenceDocumentExpired controls document-expired email.
	NotificationPreferenceDocumentExpired NotificationPreference = "DocumentExpired"
	// NotificationPreferenceDocumentExpirationReset controls expiration-reset email.
	NotificationPreferenceDocumentExpirationReset NotificationPreference = "DocumentExpirationReset"
	// NotificationPreferenceDocumentProcessingFailed controls document-failure email.
	NotificationPreferenceDocumentProcessingFailed NotificationPreference = "DocumentProcessingFailed"
	// NotificationPreferenceTemplateProcessingFailed controls template-failure email.
	NotificationPreferenceTemplateProcessingFailed NotificationPreference = "TemplateProcessingFailed"
	// NotificationPreferenceSignerWhatsAppFailed controls WhatsApp-failure email.
	NotificationPreferenceSignerWhatsAppFailed NotificationPreference = "SignerWhatsappFailed"
)

// NotificationPreferences maps notification codes to whether their email is enabled.
// Update requests may contain only the keys to change; responses contain all nine keys.
type NotificationPreferences map[NotificationPreference]bool
