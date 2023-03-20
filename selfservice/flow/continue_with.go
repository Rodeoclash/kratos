// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package flow

import (
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/ory/x/urlx"
)

// swagger:model continueWith
type ContinueWith interface {
}

// swagger:enum ContinueWithAction
type ContinueWithAction string

// #nosec G101 -- only a key constant
const (
	ContinueWithActionSetOrySessionToken ContinueWithAction = "set_ory_session_token"
	ContinueWithActionVerificationUI     ContinueWithAction = "verification_ui"
)

var _ ContinueWith = new(ContinueWithSetToken)

// Indicates that a session was issued, and the application should use this token for authenticated requests
// swagger:model continueWithSetOrySessionToken
type ContinueWithSetToken struct {
	// Action will always be `set_ory_session_token`
	//
	// required: true
	Action ContinueWithAction `json:"action"`

	// Token is the token of the session
	//
	// required: true
	OrySessionToken string `json:"ory_session_token"`
}

func (ContinueWithSetToken) AppendTo(url.Values) url.Values {
	return nil
}

func NewContinueWithSetToken(t string) *ContinueWithSetToken {
	return &ContinueWithSetToken{
		Action:          ContinueWithActionSetOrySessionToken,
		OrySessionToken: t,
	}
}

var _ ContinueWith = new(ContinueWithVerificationUI)

// Indicates, that the UI flow could be continued by showing a verification ui
//
// swagger:model continueWithVerificationUI
type ContinueWithVerificationUI struct {
	// Action will always be `verification_ui`
	//
	// required: true
	Action ContinueWithAction `json:"action"`
	// Flow contains the ID of the verification flow
	//
	// required: true
	Flow ContinueWithVerificationUIFlow `json:"flow"`
}

// swagger:model continueWithVerificationUIFlow
type ContinueWithVerificationUIFlow struct {
	// The ID of the verification flow
	//
	// required: true
	ID uuid.UUID `json:"id"`

	// The address that should be verified in this flow
	//
	// required: true
	VerifiableAddress string `json:"verifiable_address"`
}

func NewContinueWithVerificationUI(f Flow, address string) *ContinueWithVerificationUI {
	return &ContinueWithVerificationUI{
		Action: ContinueWithActionVerificationUI,
		Flow: ContinueWithVerificationUIFlow{
			ID:                f.GetID(),
			VerifiableAddress: address,
		},
	}
}

func (c ContinueWithVerificationUI) AppendTo(src *url.URL) *url.URL {
	values := src.Query()
	values.Set("flow", c.Flow.ID.String())
	return urlx.CopyWithQuery(src, values)
}

type FlowWithContinueWith interface {
	Flow
	AddContinueWith(ContinueWith)
	ContinueWith() []ContinueWith
}
