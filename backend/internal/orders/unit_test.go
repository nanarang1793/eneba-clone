package orders

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNormalizeItemsMergesDuplicates(t *testing.T) {
	got, err := normalizeItems([]CartItem{{OfferID: 1, Qty: 2}, {OfferID: 2, Qty: 1}, {OfferID: 1, Qty: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != (CartItem{OfferID: 1, Qty: 5}) || got[1] != (CartItem{OfferID: 2, Qty: 1}) {
		t.Errorf("got %+v", got)
	}
}

func TestNormalizeItemsRejectsBadCarts(t *testing.T) {
	tooMany := make([]CartItem, maxLines+1)
	for i := range tooMany {
		tooMany[i] = CartItem{OfferID: int64(i + 1), Qty: 1}
	}
	bad := map[string][]CartItem{
		"empty":               {},
		"zero quantity":       {{OfferID: 1, Qty: 0}},
		"too many keys":       {{OfferID: 1, Qty: maxQty + 1}},
		"bad offer id":        {{OfferID: 0, Qty: 1}},
		"merged over the max": {{OfferID: 1, Qty: 6}, {OfferID: 1, Qty: 5}},
		"too many lines":      tooMany,
	}
	for name, items := range bad {
		if _, err := normalizeItems(items); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSimulatedPayments(t *testing.T) {
	p := SimulatedPayments{}
	ctx := context.Background()

	ref, err := p.Charge(ctx, 1000, "4242 4242 4242 4242") // spaces are fine
	if err != nil || !strings.HasPrefix(ref, "sim_") {
		t.Errorf("success card: ref=%q err=%v", ref, err)
	}
	if _, err := p.Charge(ctx, 1000, "4000-0000-0000-0002"); !errors.Is(err, ErrDeclined) {
		t.Errorf("decline card: got %v", err)
	}
	if _, err := p.Charge(ctx, 1000, "1234"); !errors.Is(err, ErrInvalidCard) {
		t.Errorf("other card: got %v", err)
	}
}

func TestIsUUID(t *testing.T) {
	if !isUUID("02358386-fe28-4c66-a466-cf9b3930087b") {
		t.Error("valid uuid rejected")
	}
	for _, s := range []string{"", "not-a-uuid", "02358386-fe28-4c66-a466-cf9b3930087", "'; DROP TABLE orders; --"} {
		if isUUID(s) {
			t.Errorf("%q accepted", s)
		}
	}
}
