package main

import (
	"encoding/json"
	"fmt"

	perhap "github.com/version2beta/perhap/sdk/go"
)

// Cart is the model maintained by the cart domain service.
type Cart struct {
	Items    map[string]CartItem `json:"items"`     // keyed by product ID
	Subtotal float64             `json:"subtotal"`
	Tax      float64             `json:"tax"`
	Total    float64             `json:"total"`
}

type CartItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`  // unit price
	Quantity  int     `json:"quantity"`
	LineTotal float64 `json:"line_total"`
}

// Event data shapes.
type ItemAddedData struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type ItemRemovedData struct {
	ProductID string `json:"product_id"`
}

type QuantityChangedData struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// newCart returns a zero-value Cart ready for reduction.
func newCart() Cart {
	return Cart{Items: make(map[string]CartItem)}
}

// CartReducer is the pure reducer for the cart domain.
func CartReducer(eventType string, model Cart, event perhap.Event) (Cart, []perhap.Event, error) {
	if model.Items == nil {
		model.Items = make(map[string]CartItem)
	}

	switch eventType {
	case "item_added":
		var d ItemAddedData
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return model, nil, fmt.Errorf("unmarshal item_added: %w", err)
		}
		if existing, ok := model.Items[d.ProductID]; ok {
			existing.Quantity += d.Quantity
			existing.LineTotal = existing.Price * float64(existing.Quantity)
			model.Items[d.ProductID] = existing
		} else {
			model.Items[d.ProductID] = CartItem{
				ProductID: d.ProductID,
				Name:      d.Name,
				Price:     d.Price,
				Quantity:  d.Quantity,
				LineTotal: d.Price * float64(d.Quantity),
			}
		}

	case "item_removed":
		var d ItemRemovedData
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return model, nil, fmt.Errorf("unmarshal item_removed: %w", err)
		}
		delete(model.Items, d.ProductID)

	case "quantity_changed":
		var d QuantityChangedData
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return model, nil, fmt.Errorf("unmarshal quantity_changed: %w", err)
		}
		if item, ok := model.Items[d.ProductID]; ok {
			item.Quantity = d.Quantity
			item.LineTotal = item.Price * float64(d.Quantity)
			model.Items[d.ProductID] = item
		}
	}

	model = recalculate(model)
	return model, nil, nil
}

const taxRate = 0.08875 // NYC combined tax rate, for flavor

func recalculate(cart Cart) Cart {
	subtotal := 0.0
	for _, item := range cart.Items {
		subtotal += item.LineTotal
	}
	tax := subtotal * taxRate
	cart.Subtotal = round2(subtotal)
	cart.Tax = round2(tax)
	cart.Total = round2(subtotal + tax)
	return cart
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// CartValidator checks domain invariants.
func CartValidator(model Cart) []perhap.ValidationError {
	var errs []perhap.ValidationError
	for id, item := range model.Items {
		if item.Quantity <= 0 {
			errs = append(errs, perhap.ValidationError{
				Field:   fmt.Sprintf("items.%s.quantity", id),
				Message: "quantity must be positive",
			})
		}
		if item.Price < 0 {
			errs = append(errs, perhap.ValidationError{
				Field:   fmt.Sprintf("items.%s.price", id),
				Message: "price cannot be negative",
			})
		}
	}
	return errs
}
