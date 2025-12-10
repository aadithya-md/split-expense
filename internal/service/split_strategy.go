package service

import (
	"fmt"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/aadithya-md/split-expense/internal/util"
)

type SplitStrategy interface {
	CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) // Removed usersMap
}

type equalSplitStrategy struct{}

func (s *equalSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.EqualSplits) == 0 {
		return nil, fmt.Errorf("equal split requires participants")
	}

	amountPerUser := util.RoundToTwoDecimalPlaces(req.TotalAmount / float64(len(req.EqualSplits)))

	splits := make([]repository.ExpenseSplit, 0, len(req.EqualSplits))
	var currentTotalOwed float64

	for i, es := range req.EqualSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := amountPerUser
		if i == 0 { // Distribute rounding error to the first user
			splitOwed = util.RoundToTwoDecimalPlaces(req.TotalAmount - (amountPerUser * float64(len(req.EqualSplits)-1)))
		}
		splits = append(splits, repository.ExpenseSplit{
			UserID:     es.UserID, // Use pre-populated UserID
			AmountPaid: util.RoundToTwoDecimalPlaces(es.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Final check to ensure total owed matches total amount after rounding adjustments
	if util.RoundToTwoDecimalPlaces(currentTotalOwed) != util.RoundToTwoDecimalPlaces(req.TotalAmount) {
		return nil, fmt.Errorf("rounding error: sum of equal split amounts (%.2f) does not match total amount (%.2f)", currentTotalOwed, req.TotalAmount)
	}

	return splits, nil
}

type percentageSplitStrategy struct{}

func (s *percentageSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.PercentageSplits) == 0 {
		return nil, fmt.Errorf("percentage split requires percentages")
	}

	var totalPercentage float64
	for _, ps := range req.PercentageSplits {
		totalPercentage += ps.Percentage
	}
	if totalPercentage != 100 {
		return nil, fmt.Errorf("percentage split total must be 100%%")
	}

	splits := make([]repository.ExpenseSplit, 0, len(req.PercentageSplits))
	var currentTotalOwed float64

	for _, ps := range req.PercentageSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := util.RoundToTwoDecimalPlaces(req.TotalAmount * (ps.Percentage / 100))
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ps.UserID, // Use pre-populated UserID
			AmountPaid: util.RoundToTwoDecimalPlaces(ps.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Adjust for rounding errors
	diff := util.RoundToTwoDecimalPlaces(req.TotalAmount - currentTotalOwed)
	if diff != 0 && len(splits) > 0 {
		splits[0].AmountOwed = util.RoundToTwoDecimalPlaces(splits[0].AmountOwed + diff)
	}

	return splits, nil
}

type manualSplitStrategy struct{}

func (s *manualSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.ManualSplits) == 0 {
		return nil, fmt.Errorf("manual split requires manual amounts")
	}

	var totalOwed float64
	splits := make([]repository.ExpenseSplit, 0, len(req.ManualSplits))
	for _, ms := range req.ManualSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := util.RoundToTwoDecimalPlaces(ms.AmountOwed)
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ms.UserID, // Use pre-populated UserID
			AmountPaid: util.RoundToTwoDecimalPlaces(ms.AmountPaid),
			AmountOwed: splitOwed,
		})
		totalOwed += splitOwed
	}

	if util.RoundToTwoDecimalPlaces(totalOwed) != util.RoundToTwoDecimalPlaces(req.TotalAmount) {
		return nil, fmt.Errorf("manual split amounts (%.2f) must sum up to total amount (%.2f)", totalOwed, req.TotalAmount)
	}

	return splits, nil
}

func getSplitStrategy(method SplitMethodType) (SplitStrategy, error) {
	switch method {
	case SplitMethodEqual:
		return &equalSplitStrategy{}, nil
	case SplitMethodPercentage:
		return &percentageSplitStrategy{}, nil
	case SplitMethodManual:
		return &manualSplitStrategy{}, nil
	default:
		return nil, fmt.Errorf("invalid split method: %s", method)
	}
}
