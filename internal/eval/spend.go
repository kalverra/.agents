package eval

import (
	"encoding/json"
	"fmt"
	"os"
)

// ModelCost defines the price per 1M tokens for input and output.
type ModelCost struct {
	InputPrice  float64 // USD per 1M tokens
	OutputPrice float64 // USD per 1M tokens
}

var modelCosts = map[string]ModelCost{
	"gemini-2.5-flash": {InputPrice: 0.10, OutputPrice: 0.40},
	"gemini-2.5-pro":   {InputPrice: 1.25, OutputPrice: 5.00},
	// Fallback/Default for unknown models
	"default": {InputPrice: 0.15, OutputPrice: 0.60},
}

// CalculateCost returns the USD cost for a given model and token usage.
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	cost, ok := modelCosts[model]
	if !ok {
		cost = modelCosts["default"]
	}
	return (float64(inputTokens)*cost.InputPrice + float64(outputTokens)*cost.OutputPrice) / 1_000_000
}

// Spend tracks cumulative API expenditure.
type Spend struct {
	CumulativeCost float64 `json:"cumulative_cost"`
}

// LoadSpend reads eval_spend.json.
func LoadSpend(path string) (Spend, error) {
	data, err := os.ReadFile(path) //nolint: gosec
	if err != nil {
		if os.IsNotExist(err) {
			return Spend{}, nil
		}
		return Spend{}, err
	}
	var s Spend
	if err := json.Unmarshal(data, &s); err != nil {
		return Spend{}, err
	}
	return s, nil
}

// SaveSpend writes eval_spend.json.
func SaveSpend(path string, s Spend) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// CheckSpendCap returns an error if cumulative cost exceeds the cap.
func CheckSpendCap(path string, spendCap float64) error {
	spend, err := LoadSpend(path)
	if err != nil {
		return fmt.Errorf("loading spend: %w", err)
	}
	if spend.CumulativeCost >= spendCap {
		return fmt.Errorf("spending cap exceeded: $%.4f spent of $%.4f cap", spend.CumulativeCost, spendCap)
	}
	return nil
}
