package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/account"
	configSvc "github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/event"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

type config struct {
	AccountCreatedTopicArn string `env:"ACCOUNT_CREATED_TOPIC_ARN"`
	AccountUpdatedTopicArn string `env:"ACCOUNT_UPDATED_TOPIC_ARN"`
	AccountDeletedTopicArn string `env:"ACCOUNT_DELETED_TOPIC_ARN"`
}

var (
	Config       config
	statusLookup map[account.Status]AccountStatusCount
)

/*
Key												Value
LeasedAccountsCount
ReadyAccountsCount
NotReadyAccountsCount
OrphanedAccountsCount
NoneAccountsCount
TotalAccountsCount
*/

// TODO: move to data layer
type AccountStatusCount string

const (
	LeasedAccountsCount   AccountStatusCount = "LeasedAccountsCount"
	ReadyAccountsCount    AccountStatusCount = "ReadyAccountsCount"
	NotReadyAccountsCount AccountStatusCount = "NotReadyAccountsCount"
	OrphanedAccountsCount AccountStatusCount = "OrphanedAccountsCount"
	NoneAccountsCount     AccountStatusCount = "NoneAccountsCount"
	TotalAccountsCount    AccountStatusCount = "TotalAccountsCount"
)

func init() {
	cfgBldr := &configSvc.ConfigurationBuilder{}
	Config := &config{}
	if err := cfgBldr.Unmarshal(Config); err != nil {
		log.Fatalf("Could not load configuration: %s", err)
	}

	statusLookup := map[account.Status]AccountStatusCount{
		account.StatusNone:     NoneAccountsCount,
		account.StatusReady:    ReadyAccountsCount,
		account.StatusNotReady: NotReadyAccountsCount,
		account.StatusLeased:   LeasedAccountsCount,
		account.StatusOrphaned: OrphanedAccountsCount,
	}
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	// Check what kind of event we got, to pull our account object from it
	var oldAccount, newAccount *account.Account
	snsRecord := snsEvent.Records[0].SNS
	switch snsRecord.TopicArn {
	case Config.AccountCreatedTopicArn:
		oldAccount = nil
		var err error
		newAccount, err = unmarshalAccount(snsRecord.Message)
		if err != nil {
			return err
		}
	case Config.AccountDeletedTopicArn:
		newAccount = nil
		var err error
		oldAccount, err = unmarshalAccount(snsRecord.Message)
		if err != nil {
			return err
		}
	case Config.AccountUpdatedTopicArn:
		evt, err := unmarshalAccountUpdate(snsRecord.Message)
		if err != nil {
			return err
		}

		oldAccount = evt.OldAccount
		newAccount = evt.NewAccount
	default:
		return fmt.Errorf("Unexpected Topic Arn: %s", snsRecord.TopicArn)
	}

	increments := map[AccountStatusCount]int64{}

	// Adding an account
	if oldAccount == nil {
		// +1 to total accounts
		increments[TotalAccountsCount] = 1

		// +1 to <new account status>
		countToIncr := statusLookup[*newAccount.Status]
		increments[countToIncr] = 1
	}
	// Removing an account
	if newAccount == nil {
		// -1 to total accounts
		increments[TotalAccountsCount] = -1

		// -1 to <old account status>
		countToIncr := statusLookup[*oldAccount.Status]
		increments[countToIncr] = -1
	}

	// Changed account status
	didStatusChange := oldAccount != nil &&
		newAccount != nil &&
		oldAccount.Status != newAccount.Status
	if didStatusChange {
		// -1 to <old account status>
		countToIncr := statusLookup[*oldAccount.Status]
		increments[countToIncr] = -1

		// +1 to <new account status>
		countToIncr = statusLookup[*newAccount.Status]
		increments[countToIncr] = 1
	}

	// Persist increments to DB
	for key, incr := range increments {
		// TODO: implement
		err := dataSvc.IncrementAccountsCount(key, incr)

		if err != nil {
			return err
		}
	}

	// TODO:
	// - Check if it's a create, update, or delete event
	// - Map the event to increments/decrements
	// 		CreateAccount => <status> ++, TotalAccountsCount ++
	//		DeleteAccount => <status> --, TotalAccountsCount --
	// 		UpdateAccount => <prevStatus> --, <nextStatus> ++
	// - Data layer with, `IncrementAccountsCount(status string, incr int)` method

	// - Init Logic
	//	if any of the fields are missing, reinitialize from Accounts table scan
	// 		or, what if we have a LastInitialized field
	// 		and if it's empty (or older the X days),
	//		we reinitialize.
	//		this will also account for any drift
	return nil
}

func unmarshalAccount(jsonStr string) (*account.Account, error) {
	var acct account.Account
	err := json.Unmarshal([]byte(jsonStr), &acct)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal SNS messsage %s %s", jsonStr, err)
	}
	return &acct, nil
}

func unmarshalAccountUpdate(jsonStr string) (*event.AccountUpdateEvent, error) {
	var evt event.AccountUpdateEvent
	err := json.Unmarshal([]byte(jsonStr), &evt)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal SNS messsage %s %s", jsonStr, err)
	}
	return &evt, nil
}

func main() {
	lambda.Start(Handler)
}
