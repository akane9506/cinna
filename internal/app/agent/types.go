package agent

import (
	"fmt"
	"strings"
)

// For intent classification
type Intent string

const (
	IntentOther    Intent = "OTHER"
	IntentShopping Intent = "SHOPPING"
	IntentFeedback Intent = "FEEDBACK"
	IntentFailed   Intent = "FAILED"
)

// for action classification
type Action string

const (
	ActionList   Action = "LIST"
	ActionUpdate Action = "UPDATE"
	ActionNone   Action = "NONE"
)

// for db updates
type DBMethod string

const (
	MethodAdd    DBMethod = "ADD"
	MethodRemove DBMethod = "REMOVE"
	MethodModify DBMethod = "MODIFY"
)

// shopping list categories
type ShoppingCategory string

const (
	ShoppingGrocery    ShoppingCategory = "grocery"
	ShoppingPharmacy   ShoppingCategory = "pharmacy"
	ShoppingPet        ShoppingCategory = "pet"
	ShoppingToy        ShoppingCategory = "toy"
	ShoppingStationary ShoppingCategory = "stationery"
	ShoppingOther      ShoppingCategory = "other"
)

// normalize the parsed intent
func normalizeIntent(raw string) Intent {
	intent := Intent(strings.ToUpper(strings.TrimSpace(raw)))
	switch intent {
	case IntentShopping, IntentFeedback, IntentOther:
		return intent
	default:
		return IntentFailed
	}
}

// normalize the parsed action type
func normalizeAction(raw string) Action {
	action := Action(strings.ToUpper(strings.TrimSpace(raw)))
	switch action {
	case ActionList, ActionUpdate, ActionNone:
		return action
	default:
		return ActionNone
	}
}

// normalize db update method
// we need an error return for this one because the method affects the db operation
// invalid operation type will ruin the db
func normalizeMethod(raw string) (DBMethod, error) {
	method := DBMethod(strings.ToUpper(strings.TrimSpace(raw)))
	switch method {
	case MethodAdd, MethodModify, MethodRemove:
		return method, nil
	default:
		return "", fmt.Errorf("invalid method type: %s", method)
	}
}

// normalize shopping category
func normalizeShoppingCategory(raw string) ShoppingCategory {
	category := ShoppingCategory(strings.ToLower(strings.TrimSpace(raw)))
	switch category {
	case ShoppingGrocery, ShoppingPharmacy, ShoppingPet, ShoppingToy, ShoppingStationary, ShoppingOther:
		return category
	default:
		return ShoppingOther
	}
}
