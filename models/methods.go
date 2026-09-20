package models

// Verification methods for SignerReference.VerificationMethod,
// TemplateSigner.VerificationMethod and EstimateAssignmentCostSigner.
// Verification is how a signer proves who they are before signing.
//
// Verification and notification are coupled: send one, both, or neither, and
// the missing side is inferred. Sending neither defaults both to Email. Only
// matching pairs are accepted — Email verifies with Email, Whatsapp with
// Whatsapp, and DigitalCertificate with either — and an invalid pairing is a
// 400. Only one notification method is allowed per signer.
const (
	// VerificationMethodEmail emails the signer a one-time code to enter before
	// signing. It is the default and costs nothing.
	VerificationMethodEmail = "Email"
	// VerificationMethodWhatsApp sends that code over WhatsApp. It forces the
	// WhatsApp notification channel, so the signer costs that channel's 0.45
	// credits, and it needs a whatsapp_phone_number and a paid subscription.
	VerificationMethodWhatsApp = "Whatsapp"
	// VerificationMethodDigitalCertificate has the signer sign with their own
	// ICP-Brasil certificate (A1 in software, A3 on a token or smart card) through
	// the Web PKI browser extension, producing a qualified PAdES signature.
	//
	// It needs the Digital Certificate feature on the account (Standard and Pro
	// plans), a CPF or CNPJ in the signer's government_id, and the signer alone
	// in their signing step. A CPF requires that person's own certificate — an
	// e-CPF, or an e-CNPJ naming them as legal representative; a CNPJ requires an
	// e-CNPJ for that company. It costs 2 credits per signer on top of the
	// notification, charged when the assignment is created and itemized in the
	// cost breakdown under the SignatureDigitalCertificate code.
	VerificationMethodDigitalCertificate = "DigitalCertificate"
)

// Notification methods for SignerReference.NotificationMethods and its template
// and estimate equivalents. The notification is what delivers the signing
// invitation, and its cost is charged per signer when the assignment is created
// and again on every resend.
const (
	// NotificationMethodEmail emails an invitation with a link to sign. It needs
	// an email address on the signer and costs nothing.
	NotificationMethodEmail = "Email"
	// NotificationMethodWhatsApp sends that invitation over WhatsApp. It needs a
	// whatsapp_phone_number on the signer, costs 0.45 credits per signer, and is
	// available only on paid subscriptions.
	NotificationMethodWhatsApp = "Whatsapp"
)
