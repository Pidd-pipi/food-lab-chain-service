package domain

import "errors"

var ErrSpecimenNotFound = errors.New("specimen not found")

func Handoff(specimen *Specimen, recipient string) error {
	if specimen.ChainState == "sealed" {
		return errors.New("sealed specimen cannot be handed off")
	}
	if recipient == "" {
		return errors.New("recipient is required")
	}
	specimen.Custodian = recipient
	specimen.LastHandoff = recipient
	specimen.ChainState = "in-transit"
	return nil
}
