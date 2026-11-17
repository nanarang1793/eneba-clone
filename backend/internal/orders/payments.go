package orders

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

// PaymentProvider is what checkout needs from a payment company.
// The app only depends on this interface, so the simulated provider below
// could be replaced by a Stripe one without touching the order logic.
type PaymentProvider interface {
	// Charge takes amountCents from the card. It returns the provider's
	// payment reference, ErrDeclined, or ErrInvalidCard.
	Charge(ctx context.Context, amountCents int, cardNumber string) (ref string, err error)
}

var (
	ErrDeclined    = errors.New("card declined")
	ErrInvalidCard = errors.New("invalid card")
)

// Test cards, like Stripe's test mode:
const (
	TestCardSuccess  = "4242424242424242"
	TestCardDeclined = "4000000000000002"
)

// SimulatedPayments approves 4242 4242 4242 4242, declines
// 4000 0000 0000 0002 and rejects anything else. No money moves.
type SimulatedPayments struct{}

func (SimulatedPayments) Charge(_ context.Context, amountCents int, cardNumber string) (string, error) {
	card := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")
	switch card {
	case TestCardSuccess:
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		return "sim_" + hex.EncodeToString(b), nil
	case TestCardDeclined:
		return "", ErrDeclined
	default:
		return "", ErrInvalidCard
	}
}
