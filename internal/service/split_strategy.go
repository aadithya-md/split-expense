package service

import (
	"fmt"
	"math"

	"github.com/aadithya-md/split-expense/internal/repository"
)

// roundToTwoDecimalPlaces rounds a float64 to two decimal places.
func roundToTwoDecimalPlaces(f float64) float64 {
	return math.Round(f*100) / 100
}

type SplitStrategy interface {
	CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) // Removed usersMap
}

type equalSplitStrategy struct{}

func (s *equalSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.EqualSplits) == 0 {
		return nil, fmt.Errorf("equal split requires participants")
	}

	amountPerUser := roundToTwoDecimalPlaces(req.TotalAmount / float64(len(req.EqualSplits)))

	splits := make([]repository.ExpenseSplit, 0, len(req.EqualSplits))
	var currentTotalOwed float64

	for i, es := range req.EqualSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := amountPerUser
		if i == 0 { // Distribute rounding error to the first user
			splitOwed = roundToTwoDecimalPlaces(req.TotalAmount - (amountPerUser * float64(len(req.EqualSplits)-1)))
		}
		splits = append(splits, repository.ExpenseSplit{
			UserID:     es.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(es.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Final check to ensure total owed matches total amount after rounding adjustments
	if roundToTwoDecimalPlaces(currentTotalOwed) != roundToTwoDecimalPlaces(req.TotalAmount) {
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
		splitOwed := roundToTwoDecimalPlaces(req.TotalAmount * (ps.Percentage / 100))
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ps.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(ps.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Adjust for rounding errors
	diff := roundToTwoDecimalPlaces(req.TotalAmount - currentTotalOwed)
	if diff != 0 && len(splits) > 0 {
		splits[0].AmountOwed = roundToTwoDecimalPlaces(splits[0].AmountOwed + diff)
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
		splitOwed := roundToTwoDecimalPlaces(ms.AmountOwed)
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ms.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(ms.AmountPaid),
			AmountOwed: splitOwed,
		})
		totalOwed += splitOwed
	}

	if roundToTwoDecimalPlaces(totalOwed) != roundToTwoDecimalPlaces(req.TotalAmount) {
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
