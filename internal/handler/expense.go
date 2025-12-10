package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aadithya-md/split-expense/internal/service"
	"github.com/aadithya-md/split-expense/internal/util"
	"github.com/gorilla/mux"
)

type ExpenseHandler struct {
	expenseService service.ExpenseService
}

func NewExpenseHandler(expenseService service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseService: expenseService}
}

func (h *ExpenseHandler) CreateExpenseHandler(w http.ResponseWriter, r *http.Request) {
	var req service.CreateExpenseRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validateCreateExpenseRequest(req); err != nil {
		http.Error(w, "Invalid expense data: "+err.Error(), http.StatusBadRequest)
		return
	}

	expense, err := h.expenseService.CreateExpense(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) GetExpensesForUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userEmail := vars["email"]
	if userEmail == "" {
		http.Error(w, "User email is required", http.StatusBadRequest)
		return
	}

	expenses, err := h.expenseService.GetExpensesForUser(userEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expenses)
}

func (h *ExpenseHandler) validateCreateExpenseRequest(req service.CreateExpenseRequest) error {
	if req.Description == "" || req.TotalAmount <= 0 || req.CreatedByEmail == "" || req.SplitMethod == "" {
		return fmt.Errorf("description, total_amount, created_by, and split_method are required")
	}

	// Validate unique emails
	participatingEmails := util.NewSet[string]()

	switch req.SplitMethod {
	case service.SplitMethodEqual:
		if len(req.EqualSplits) == 0 {
			return fmt.Errorf("equal split requires participants with amounts paid")
		}
		for _, s := range req.EqualSplits {
			if participatingEmails.IsMember(s.UserEmail) {
				return fmt.Errorf("duplicate email found in splits: %s", s.UserEmail)
			}
			participatingEmails.Add(s.UserEmail)

		}
	case service.SplitMethodPercentage:
		if len(req.PercentageSplits) == 0 {
			return fmt.Errorf("percentage split requires percentages")
		}
		var totalPercentage float64
		for _, s := range req.PercentageSplits {
			if participatingEmails.IsMember(s.UserEmail) {
				return fmt.Errorf("duplicate email found in percentage splits: %s", s.UserEmail)
			}
			participatingEmails.Add(s.UserEmail)
			totalPercentage += s.Percentage
		}
		if totalPercentage != 100 {
			return fmt.Errorf("total percentage across all splits must be 100%%")
		}
	case service.SplitMethodManual:
		if len(req.ManualSplits) == 0 {
			return fmt.Errorf("manual split requires manual amounts")
		}
		var totalOwed float64
		for _, s := range req.ManualSplits {
			if participatingEmails.IsMember(s.UserEmail) {
				return fmt.Errorf("duplicate email found in manual splits: %s", s.UserEmail)
			}
			participatingEmails.Add(s.UserEmail)
			totalOwed += s.AmountOwed
		}
		if totalOwed != req.TotalAmount {
			return fmt.Errorf("total amount owed across all splits (%.2f) does not match total expense amount (%.2f)", totalOwed, req.TotalAmount)
		}
	default:
		return fmt.Errorf("unsupported split method")
	}

	if !participatingEmails.IsMember(req.CreatedByEmail) {

		return fmt.Errorf("created_by user (%s) must be included in the split participants", req.CreatedByEmail)
	}

	return nil
}

func (h *ExpenseHandler) GetOutstandingBalancesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userEmail := vars["email"]
	if userEmail == "" {
		http.Error(w, "User email is required", http.StatusBadRequest)
		return
	}

	balances, err := h.expenseService.GetOutstandingBalancesForUser(userEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(balances)
}
