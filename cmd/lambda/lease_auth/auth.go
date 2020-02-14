package main

import (
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/account/accountiface"
	"github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/errors"
	leases "github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface"
	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func LeaseAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve the API Gateway context from our HTTP request object
	// (required for our UserDetails service, to extract Cognito info)
	reqCtx, err := muxLambda.GetAPIGatewayContext(r)
	if err != nil {
		log.Printf("Failed to parse context object from request: %s", err)
		api.WriteAPIErrorResponse(w,
			errors.NewInternalServer("Internal server error", err),
		)
		return
	}

	// Lease ID is optional.
	// If not provided, we will attempt to find a matching
	// lease, where PrincipalID == Cognito User ID
	var leaseID *string
	muxVars := mux.Vars(r)
	if muxVars != nil {
		leaseIDStr, ok := muxVars["leaseID"]
		if ok {
			leaseID = &leaseIDStr
		}
	}

	// Generate credentials for the requested lease
	res, err := leaseAuth(&leaseAuthInput{
		leaseID:        leaseID,
		requestContext: &reqCtx,
		leaseService:   Services.LeaseService(),
		accountService: Services.AccountService(),
		userDetailer:   Services.UserDetailer(),
		accountManager: Services.AccountManager(),
		// Note that AWS has a hard 1-hour limit on roles assumed
		// via "role-chaining" (assuming a role from an assumed-role).
		// In our case, we're assuming a role, via our assumed lambda execution role.
		principalSessionDuration: time.Hour,
	})
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, 201, res)
}

type leaseAuthInput struct {
	leaseID                  *string
	requestContext           *events.APIGatewayProxyRequestContext
	leaseService             leaseiface.Servicer
	accountService           accountiface.Servicer
	userDetailer             api.UserDetailer
	accountManager           accountmanageriface.Servicer
	principalSessionDuration time.Duration
}

func leaseAuth(input *leaseAuthInput) (*response.LeaseAuthResponse, error) {
	// Lookup the requesting user, via cognito
	user := input.userDetailer.GetUser(input.requestContext)

	var lease *leases.Lease
	var err error
	if input.leaseID != nil {
		// Grab the lease from the DB
		lease, err = input.leaseService.Get(*input.leaseID)
		if err != nil {
			return nil, err
		}
	} else if user.Username != "" {
		// Lookup the active lease for the requesting user
		leaseList, err := input.leaseService.List(&leases.Lease{
			PrincipalID: &user.Username,
			Status:      leases.StatusActive.StatusPtr(),
		})
		if err != nil {
			return nil, err
		}
		leaseListSlice := []leases.Lease(*leaseList)
		if len(leaseListSlice) >= 1 {
			lease = &leaseListSlice[0]
		}
	}
	if lease == nil {
		return nil, errors.NewNotFound("lease for user", user.Username)
	}

	// Return auth error, if lease isn't active
	if *lease.Status != leases.StatusActive {
		return nil, errors.NewUnauthorized("Unable to authorize against non-active lease")
	}

	// Admin users may login to any active lease.
	// Other users can only login to their own leases
	isLoginAllowed := user.Role == api.AdminGroupName ||
		user.Username == *lease.PrincipalID
	if !isLoginAllowed {
		return nil, errors.NewUnauthorizedf("User %s does not have access to lease %s",
			user.Username, *lease.ID)
	}

	// Lookup the Account, so we can get the Principal Role ARN
	acct, err := input.accountService.Get(*lease.AccountID)
	if err != nil {
		// Return a 500 if account is missing (system is in a corrupt state, not a client error)
		if errors.NewNotFound("account", *lease.AccountID).IsStatusError(err) {
			return nil, errors.NewInternalServer("Account record is missing for the requested lease", err)
		}
		return nil, err
	}

	// Grab STS credentials for the account's PrincipalRole ARN
	roleSessionName := user.Username
	if roleSessionName == "" {
		roleSessionName = *lease.PrincipalID
	}
	creds := input.accountManager.Credentials(acct.PrincipalRoleArn, roleSessionName, &input.principalSessionDuration)
	credsValue, err := creds.Get()
	if err != nil {
		log.Printf("Failed to login to %s: %s", acct.PrincipalRoleArn, err)
		return nil, errors.NewInternalServer(
			fmt.Sprintf("Failed to assume role %s", acct.PrincipalRoleArn),
			err,
		)
	}

	// Generate a URL for logging into the AWS Web Console
	consoleURL, err := input.accountManager.ConsoleURL(creds)
	if err != nil {
		log.Printf("Failed to generate console URL for %s: %s", acct.PrincipalRoleArn, err)
		return nil, errors.NewInternalServer(
			fmt.Sprintf("Failed to generate console URL for %s", acct.PrincipalRoleArn),
			err,
		)
	}

	// Log the access key id, and the name of the authenticated user we delivered the creds to.
	// This may be used for auditing purposes, eg. to track CloudTrail events
	// back to an authenticated DCE user.
	credsMetaData := map[string]string{
		"accessKeyId": credsValue.AccessKeyID,
		"principalId": *lease.PrincipalID,
		// Note that the username will be an empty string, for requests
		// not authenticated via Cognito
		"username": user.Username,
	}
	credsLog, err := json.Marshal(credsMetaData)
	if err != nil {
		log.Printf("WARNING: Failed to log lease auth data (%v): %s", credsMetaData, err)
	} else {
		log.Println(credsLog)
	}

	// Grab the credentials expires time
	expiresOn, err := creds.ExpiresAt()
	if err != nil {
		log.Println(err)
		return nil, errors.NewInternalServer("Internal server error", err)
	}

	return &response.LeaseAuthResponse{
		AccessKeyID:     credsValue.AccessKeyID,
		SecretAccessKey: credsValue.SecretAccessKey,
		SessionToken:    credsValue.SessionToken,
		ConsoleURL:      consoleURL,
		ExpiresOn:       expiresOn.Unix(),
	}, nil
}
