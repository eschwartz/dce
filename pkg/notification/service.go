package notification

import "github.com/Optum/dce/pkg/lease"

type Notificationer interface {
	BudgetNotification(input *BudgetNotificationInput) error
}


type Notification struct {
	principalBudget float64

	notificationThresholds []float64

	budgetEmailTemplates string
	/*...*/
}

type BudgetNotificationInput struct {
	activeLease lease.Lease
	principalUsage float64
	leaseUsage float64
}

func (n *Notification) BudgetNotification(input *BudgetNotificationInput) error {
	return nil
}

/*
struct {
		// DEPRECATED
		// Populate these, but modify default templates
		// to use the new template vars
		ActualSpend         float64
		IsOverBudget        bool
		ThresholdPercentile int

		// Active lease
		Lease               db.Lease

		// New
		LeaseUsage
		LeaseBudget

		PrincipalUsage
		PrincipalBudget

		LeaseBudgetPercentile
		PrincipalBudgetPercentile
	}

type BudgetNotificationer interface {
	LeaseBudget(lease *lease.Lease, actualSpend float)
	PrincipalBudget(principalID string, actualSpend float)
}

func (n *BudgetNotificationer) LeaseBudget(lease *lease.Lease, leaseSpend) error {
	// Find threshold hit against lease.Budget
	// compare to configured notification thresholds
	// Render configured templates, and send email
}


func (n *BudgetNotificationer) PrincipalBudget(principalID string, actualSpend) error {
	// Lookup active lease for the principal (needed for template)
	//
}


ACTIONS:

- we need new configurations, for principal budget email templates
*/
