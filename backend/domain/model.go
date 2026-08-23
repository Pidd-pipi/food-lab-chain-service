package domain

type Specimen struct {
	ID          string `json:"id"`
	SampleCode  string `json:"sampleCode"`
	FoodType    string `json:"foodType"`
	CollectedAt string `json:"collectedAt"`
	Custodian   string `json:"custodian"`
	ChainState  string `json:"chainState"`
	LastHandoff string `json:"lastHandoff"`
}

type HandoffRequest struct {
	To string `json:"to"`
}
