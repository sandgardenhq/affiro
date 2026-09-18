package client

// HumanWritten is the API's verdict on how likely a document was typed by a person.
type HumanWritten string

const (
	HumanWrittenLikely       HumanWritten = "likely"
	HumanWrittenPossible     HumanWritten = "possible"
	HumanWrittenUnlikely     HumanWritten = "unlikely"
	HumanWrittenVeryUnlikely HumanWritten = "very unlikely"
	HumanWrittenUnknown      HumanWritten = "unknown"
)

// Analysis is what the API made of an uploaded document.
type Analysis struct {
	ID           string       `json:"id"`
	DocumentID   string       `json:"documentId"`
	ExternalID   string       `json:"externalId"`
	HumanWritten HumanWritten `json:"humanWritten"`
	Descriptions []string     `json:"descriptions"`
}
