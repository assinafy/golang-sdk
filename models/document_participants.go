package models

// SignerCapacity is the sender-declared capacity printed on the certificate.
// It is not shown in invitations or the signing page. Empty or null selects Sign.
type SignerCapacity string

const (
	// SignerCapacitySign selects the Sign capacity.
	SignerCapacitySign SignerCapacity = "Sign"
	// SignerCapacityApprove selects the Approve capacity.
	SignerCapacityApprove SignerCapacity = "Approve"
	// SignerCapacityAcknowledge selects the Acknowledge capacity.
	SignerCapacityAcknowledge SignerCapacity = "Acknowledge"
	// SignerCapacityParty selects the Party capacity.
	SignerCapacityParty SignerCapacity = "Party"
	// SignerCapacityWitness selects the Witness capacity.
	SignerCapacityWitness SignerCapacity = "Witness"
	// SignerCapacityIntervening selects the Intervening capacity.
	SignerCapacityIntervening SignerCapacity = "Intervening"
	// SignerCapacityReceipt selects the Receipt capacity.
	SignerCapacityReceipt SignerCapacity = "Receipt"
	// SignerCapacityIssuerEndorserGuarantor selects the IssuerEndorserGuarantor capacity.
	SignerCapacityIssuerEndorserGuarantor SignerCapacity = "IssuerEndorserGuarantor"
	// SignerCapacityIssuerEndorserGuarantorSurety selects the IssuerEndorserGuarantorSurety capacity.
	SignerCapacityIssuerEndorserGuarantorSurety SignerCapacity = "IssuerEndorserGuarantorSurety"
	// SignerCapacitySurety selects the Surety capacity.
	SignerCapacitySurety SignerCapacity = "Surety"
	// SignerCapacityPartySurety selects the PartySurety capacity.
	SignerCapacityPartySurety SignerCapacity = "PartySurety"
	// SignerCapacityJointDebtor selects the JointDebtor capacity.
	SignerCapacityJointDebtor SignerCapacity = "JointDebtor"
	// SignerCapacityPartyJointDebtor selects the PartyJointDebtor capacity.
	SignerCapacityPartyJointDebtor SignerCapacity = "PartyJointDebtor"
	// SignerCapacityContractingParty selects the ContractingParty capacity.
	SignerCapacityContractingParty SignerCapacity = "ContractingParty"
	// SignerCapacityContractedParty selects the ContractedParty capacity.
	SignerCapacityContractedParty SignerCapacity = "ContractedParty"
	// SignerCapacityLandlord selects the Landlord capacity.
	SignerCapacityLandlord SignerCapacity = "Landlord"
	// SignerCapacityTenant selects the Tenant capacity.
	SignerCapacityTenant SignerCapacity = "Tenant"
	// SignerCapacityBuyer selects the Buyer capacity.
	SignerCapacityBuyer SignerCapacity = "Buyer"
	// SignerCapacitySeller selects the Seller capacity.
	SignerCapacitySeller SignerCapacity = "Seller"
	// SignerCapacityAttorneyInFact selects the AttorneyInFact capacity.
	SignerCapacityAttorneyInFact SignerCapacity = "AttorneyInFact"
	// SignerCapacityLegalRepresentative selects the LegalRepresentative capacity.
	SignerCapacityLegalRepresentative SignerCapacity = "LegalRepresentative"
	// SignerCapacityEmployer selects the Employer capacity.
	SignerCapacityEmployer SignerCapacity = "Employer"
	// SignerCapacityEmployee selects the Employee capacity.
	SignerCapacityEmployee SignerCapacity = "Employee"
)

// DocumentSignerCapacity identifies one assignment signer's declared capacity.
type DocumentSignerCapacity struct {
	// SignerID identifies the assignment participant.
	SignerID string `json:"signer_id"`
	// SignAs is the capacity fixed when the assignment was created.
	SignAs SignerCapacity `json:"sign_as"`
}

// SignerNotificationDelivery adds delivery and open times to a notification.
// An open is a hint, not proof that a person read the message.
type SignerNotificationDelivery struct {
	// SignerNotification contains the event, status, failure and send times.
	SignerNotification
	// DeliveredAt is the delivery time, or null until reported.
	DeliveredAt *Timestamp `json:"delivered_at"`
	// OpenedAt is the first email open/click or the reported WhatsApp read time.
	OpenedAt *Timestamp `json:"opened_at"`
}

// SignerNotificationHistory groups a document's delivery history by signer.
type SignerNotificationHistory struct {
	// SignerID identifies the assignment participant.
	SignerID string `json:"signer_id"`
	// NotificationHistory contains that signer's tracked deliveries.
	NotificationHistory []SignerNotificationDelivery `json:"notification_history"`
}

// SetDocumentWatchersRequest replaces contacts notified when a document ends.
type SetDocumentWatchersRequest struct {
	// WatcherIDs contains at most ten workspace signer IDs with email addresses.
	// Send an empty non-nil slice to remove every watcher; nil encodes null.
	WatcherIDs []string `json:"watcher_ids"`
}
