// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package oidc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/gofrs/uuid"

	"github.com/ory/kratos/session"

	"github.com/ory/kratos/ui/node"
	"github.com/ory/x/decoderx"
	"github.com/ory/x/sqlcon"

	"github.com/ory/kratos/selfservice/flow/registration"

	"github.com/ory/kratos/text"

	"github.com/ory/kratos/continuity"

	"github.com/pkg/errors"

	"github.com/ory/herodot"

	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/x"
)

var _ login.Strategy = new(Strategy)

func (s *Strategy) RegisterLoginRoutes(r *x.RouterPublic) {
	s.setRoutes(r)
}

func (s *Strategy) PopulateLoginMethod(r *http.Request, requestedAAL identity.AuthenticatorAssuranceLevel, l *login.Flow) error {
	// This strategy can only solve AAL1
	if requestedAAL > identity.AuthenticatorAssuranceLevel1 {
		return nil
	}

	return s.populateMethod(r, l.UI, text.NewInfoLoginWith)
}

// Update Login Flow with OpenID Connect Method
//
// swagger:model updateLoginFlowWithOidcMethod
type UpdateLoginFlowWithOidcMethod struct {
	// The provider to register with
	//
	// required: true
	Provider string `json:"provider"`

	// The CSRF Token
	CSRFToken string `json:"csrf_token"`

	// Method to use
	//
	// This field must be set to `oidc` when using the oidc method.
	//
	// required: true
	Method string `json:"method"`

	// The identity traits. This is a placeholder for the registration flow.
	Traits json.RawMessage `json:"traits"`

	// UpstreamParameters are the parameters that are passed to the upstream identity provider.
	//
	// These parameters are optional and depend on what the upstream identity provider supports.
	// Supported parameters are:
	// - `login_hint` (string): The `login_hint` parameter suppresses the account chooser and either pre-fills the email box on the sign-in form, or selects the proper session.
	// - `hd` (string): The `hd` parameter limits the login/registration process to a Google Organization, e.g. `mycollege.edu`.
	//
	// required: false
	UpstreamParameters json.RawMessage `json:"upstream_parameters"`

	// Only used in API-type flows, when an id token has been received by mobile app directly from oidc provider.
	//
	// required: false
	IDToken string `json:"id_token"`
}

func (s *Strategy) processLogin(w http.ResponseWriter, r *http.Request, a *login.Flow, token *oauth2.Token, claims *Claims, provider Provider, container *authCodeContainer) (*registration.Flow, error) {
	i, c, err := s.d.PrivilegedIdentityPool().FindByCredentialsIdentifier(r.Context(), identity.CredentialsTypeOIDC, identity.OIDCUniqueID(provider.Config().ID, claims.Subject))
	if err != nil {
		if errors.Is(err, sqlcon.ErrNoRows) {
			// If no account was found we're "manually" creating a new registration flow and redirecting the browser
			// to that endpoint.

			// That will execute the "pre registration" hook which allows to e.g. disallow this request. The registration
			// ui however will NOT be shown, instead the user is directly redirected to the auth path. That should then
			// do a silent re-request. While this might be a bit excessive from a network perspective it should usually
			// happen without any downsides to user experience as the flow has already been authorized and should
			// not need additional consent/login.

			// This is kinda hacky but the only way to ensure seamless login/registration flows when using OIDC.
			s.d.Logger().WithField("provider", provider.Config().ID).WithField("subject", claims.Subject).Debug("Received successful OpenID Connect callback but user is not registered. Re-initializing registration flow now.")

			// If return_to was set before, we need to preserve it.
			var opts []registration.FlowOption
			if len(a.ReturnTo) > 0 {
				opts = append(opts, registration.WithFlowReturnTo(a.ReturnTo))
			}

			// This flow only works for browsers anyways.
			query := r.URL.Query()
			query.Set("return_to", a.ReturnTo)
			r.URL.RawQuery = query.Encode()
			aa, err := s.d.RegistrationHandler().NewRegistrationFlow(w, r, flow.TypeBrowser, opts...)
			if err != nil {
				return nil, s.handleError(w, r, a, provider.Config().ID, nil, err)
			}

			aa.RequestURL, err = x.TakeOverReturnToParameter(a.RequestURL, aa.RequestURL)
			if err != nil {
				return nil, s.handleError(w, r, a, provider.Config().ID, nil, err)
			}

			if _, err := s.processRegistration(w, r, aa, token, claims, provider, container); err != nil {
				return aa, err
			}

			return nil, nil
		}

		return nil, s.handleError(w, r, a, provider.Config().ID, nil, err)
	}

	var o identity.CredentialsOIDC
	if err := json.NewDecoder(bytes.NewBuffer(c.Config)).Decode(&o); err != nil {
		return nil, s.handleError(w, r, a, provider.Config().ID, nil, errors.WithStack(herodot.ErrInternalServerError.WithReason("The password credentials could not be decoded properly").WithDebug(err.Error())))
	}

	sess := session.NewInactiveSession()
	sess.CompletedLoginFor(s.ID(), identity.AuthenticatorAssuranceLevel1)
	for _, c := range o.Providers {
		if c.Subject == claims.Subject && c.Provider == provider.Config().ID {
			if err = s.d.LoginHookExecutor().PostLoginHook(w, r, node.OpenIDConnectGroup, a, i, sess); err != nil {
				return nil, s.handleError(w, r, a, provider.Config().ID, nil, err)
			}
			return nil, nil
		}
	}

	return nil, s.handleError(w, r, a, provider.Config().ID, nil, errors.WithStack(herodot.ErrInternalServerError.WithReason("Unable to find matching OpenID Connect Credentials.").WithDebugf(`Unable to find credentials that match the given provider "%s" and subject "%s".`, provider.Config().ID, claims.Subject)))
}

func (s *Strategy) Login(w http.ResponseWriter, r *http.Request, f *login.Flow, identityID uuid.UUID) (i *identity.Identity, err error) {
	if err := login.CheckAAL(f, identity.AuthenticatorAssuranceLevel1); err != nil {
		return nil, err
	}

	var pid = ""
	var idToken = ""

	var p UpdateLoginFlowWithOidcMethod
	if f.Type == flow.TypeBrowser {
		if err := s.newLinkDecoder(&p, r); err != nil {
			return nil, s.handleError(w, r, f, "", nil, errors.WithStack(herodot.ErrBadRequest.WithDebug(err.Error()).WithReasonf("Unable to parse HTTP form request: %s", err.Error())))
		}
		pid = p.Provider // this can come from both url query and post body
	} else {
		if err := flow.MethodEnabledAndAllowedFromRequest(r, s.ID().String(), s.d); err != nil {
			return nil, err
		}

		if err := s.dec.Decode(r, &p,
			decoderx.HTTPDecoderSetValidatePayloads(true),
			decoderx.MustHTTPRawJSONSchemaCompiler(loginSchema),
			decoderx.HTTPDecoderJSONFollowsFormFormat()); err != nil {
			return nil, s.handleError(w, r, f, "", nil, err)
		}

		idToken = p.IDToken
		pid = p.Provider
	}

	if pid == "" {
		return nil, errors.WithStack(flow.ErrStrategyNotResponsible)
	}

	if err := flow.MethodEnabledAndAllowed(r.Context(), s.SettingsStrategyID(), s.SettingsStrategyID(), s.d); err != nil {
		return nil, s.handleError(w, r, f, pid, nil, err)
	}

	provider, err := s.provider(r.Context(), r, pid)
	if err != nil {
		return nil, s.handleError(w, r, f, pid, nil, err)
	}

	c, err := provider.OAuth2(r.Context())
	if err != nil {
		return nil, s.handleError(w, r, f, pid, nil, err)
	}

	req, err := s.validateFlow(r.Context(), r, f.ID)
	if err != nil {
		return nil, s.handleError(w, r, f, pid, nil, err)
	}

	if s.alreadyAuthenticated(w, r, req) {
		return
	}

	state := generateState(f.ID.String())
	if f.Type == flow.TypeBrowser {
		if err := s.d.ContinuityManager().Pause(r.Context(), w, r, sessionName,
			continuity.WithPayload(&authCodeContainer{
				State:  state,
				FlowID: f.ID.String(),
				Traits: p.Traits,
			}),
			continuity.WithLifespan(time.Minute*30)); err != nil {
			return nil, s.handleError(w, r, f, pid, nil, err)
		}

		f.Active = s.ID()
		if err = s.d.LoginFlowPersister().UpdateLoginFlow(r.Context(), f); err != nil {
			return nil, s.handleError(w, r, f, pid, nil, errors.WithStack(herodot.ErrInternalServerError.WithReason("Could not update flow").WithDebug(err.Error())))
		}

		var up map[string]string
		if err := json.NewDecoder(bytes.NewBuffer(p.UpstreamParameters)).Decode(&up); err != nil {
			return nil, err
		}

		codeURL := c.AuthCodeURL(state, append(provider.AuthCodeURLOptions(req), UpstreamParameters(provider, up)...)...)
		if x.IsJSONRequest(r) {
			s.d.Writer().WriteError(w, r, flow.NewBrowserLocationChangeRequiredError(codeURL))
		} else {
			http.Redirect(w, r, codeURL, http.StatusSeeOther)
		}

		return nil, errors.WithStack(flow.ErrCompletedByStrategy)
	} else if f.Type == flow.TypeAPI {
		var claims *Claims
		if apiFlowProvider, ok := provider.(APIFlowProvider); ok {
			if len(idToken) > 0 {
				claims, err = apiFlowProvider.ClaimsFromIDToken(r.Context(), idToken)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			} else {
				return nil, s.handleError(w, r, f, p.Provider, nil, ErrIDTokenMissing)
			}
		} else {
			return nil, s.handleError(w, r, f, p.Provider, nil, ErrProviderNoAPISupport)
		}

		i, _, err := s.d.PrivilegedIdentityPool().FindByCredentialsIdentifier(r.Context(),
			identity.CredentialsTypeOIDC,
			identity.OIDCUniqueID(provider.Config().ID, claims.Subject))
		if err != nil {
			if errors.Is(err, sqlcon.ErrNoRows) {
				return nil, s.handleError(w, r, f, p.Provider, nil, NewUserNotFoundError())
			}
			return nil, err
		}
		return i, nil
	} else {
		return nil, errors.WithStack(errors.New(fmt.Sprintf("Not supported flow type: %s", f.Type)))
	}
}
