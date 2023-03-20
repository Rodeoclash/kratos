/*
 * Ory Identities API
 *
 * This is the API specification for Ory Identities with features such as registration, login, recovery, account verification, profile settings, password reset, identity management, session management, email and sms delivery, and more.
 *
 * API version:
 * Contact: office@ory.sh
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
)

// ContinueWithVerificationUIFlow struct for ContinueWithVerificationUIFlow
type ContinueWithVerificationUIFlow struct {
	// The ID of the verification flow
	Id string `json:"id"`
	// The address that should be verified in this flow
	VerifiableAddress string `json:"verifiable_address"`
}

// NewContinueWithVerificationUIFlow instantiates a new ContinueWithVerificationUIFlow object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewContinueWithVerificationUIFlow(id string, verifiableAddress string) *ContinueWithVerificationUIFlow {
	this := ContinueWithVerificationUIFlow{}
	this.Id = id
	this.VerifiableAddress = verifiableAddress
	return &this
}

// NewContinueWithVerificationUIFlowWithDefaults instantiates a new ContinueWithVerificationUIFlow object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewContinueWithVerificationUIFlowWithDefaults() *ContinueWithVerificationUIFlow {
	this := ContinueWithVerificationUIFlow{}
	return &this
}

// GetId returns the Id field value
func (o *ContinueWithVerificationUIFlow) GetId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *ContinueWithVerificationUIFlow) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value
func (o *ContinueWithVerificationUIFlow) SetId(v string) {
	o.Id = v
}

// GetVerifiableAddress returns the VerifiableAddress field value
func (o *ContinueWithVerificationUIFlow) GetVerifiableAddress() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.VerifiableAddress
}

// GetVerifiableAddressOk returns a tuple with the VerifiableAddress field value
// and a boolean to check if the value has been set.
func (o *ContinueWithVerificationUIFlow) GetVerifiableAddressOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.VerifiableAddress, true
}

// SetVerifiableAddress sets field value
func (o *ContinueWithVerificationUIFlow) SetVerifiableAddress(v string) {
	o.VerifiableAddress = v
}

func (o ContinueWithVerificationUIFlow) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.Id
	}
	if true {
		toSerialize["verifiable_address"] = o.VerifiableAddress
	}
	return json.Marshal(toSerialize)
}

type NullableContinueWithVerificationUIFlow struct {
	value *ContinueWithVerificationUIFlow
	isSet bool
}

func (v NullableContinueWithVerificationUIFlow) Get() *ContinueWithVerificationUIFlow {
	return v.value
}

func (v *NullableContinueWithVerificationUIFlow) Set(val *ContinueWithVerificationUIFlow) {
	v.value = val
	v.isSet = true
}

func (v NullableContinueWithVerificationUIFlow) IsSet() bool {
	return v.isSet
}

func (v *NullableContinueWithVerificationUIFlow) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableContinueWithVerificationUIFlow(val *ContinueWithVerificationUIFlow) *NullableContinueWithVerificationUIFlow {
	return &NullableContinueWithVerificationUIFlow{value: val, isSet: true}
}

func (v NullableContinueWithVerificationUIFlow) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableContinueWithVerificationUIFlow) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
