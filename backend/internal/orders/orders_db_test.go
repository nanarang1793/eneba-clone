package orders

// Integration tests for the part that must never go wrong: a key can't be
// sold twice. They need a migrated, seeded Postgres in TEST_DATABASE_URL
// (`docker compose run --rm test`); without it they're skipped.
//
// Each test builds its own little world (a seller, one offer with N keys,
// some users) and deletes it afterwards, so the seed data is never touched.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"eneba/backend/internal/testdb"
)

type fixture struct {
	t       *testing.T
	pool    *pgxpool.Pool
	store   *Store
	offerID int64
	users   []string
}

// newFixture creates an offer with `keys` available keys and `users` accounts.
func newFixture(t *testing.T, keys, users int) *fixture {
	t.Helper()
	pool := testdb.Open(t)
	ctx := context.Background()
	tag := testdb.Tag()
	f := &fixture{t: t, pool: pool, store: NewStore(pool, SimulatedPayments{})}

	// The offer is attached to an existing product; paying bumps that
	// product's sales_count, so remember it and put it back afterwards.
	var productID int64
	var salesBefore int
	if err := pool.QueryRow(ctx, `SELECT id, sales_count FROM products ORDER BY id LIMIT 1`).Scan(&productID, &salesBefore); err != nil {
		t.Fatal(err)
	}

	var sellerID int
	err := pool.QueryRow(ctx, `INSERT INTO sellers (slug, name, rating) VALUES ($1, $1, 4.5) RETURNING id`,
		"test-seller-"+tag).Scan(&sellerID)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `
        INSERT INTO offers (product_id, seller_id, price_cents)
        VALUES ($1, $2, 1000) RETURNING id`, productID, sellerID).Scan(&f.offerID)
	if err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, pool, `
        INSERT INTO product_keys (offer_id, code)
        SELECT $1, 'TEST-' || $2 || '-' || n FROM generate_series(1, $3) n`, f.offerID, tag, keys)
	for i := 0; i < users; i++ {
		var id string
		err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, display_name) VALUES ($1, 'x', 'Test') RETURNING id`,
			fmt.Sprintf("buyer%d-%s@test.example", i, tag)).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		f.users = append(f.users, id)
	}

	// Clean up in foreign-key order: keys point at order lines, lines at
	// orders and offers, orders at users.
	t.Cleanup(func() {
		testdb.Exec(t, pool, `DELETE FROM product_keys WHERE offer_id = $1`, f.offerID)
		testdb.Exec(t, pool, `DELETE FROM orders WHERE user_id = ANY($1::uuid[])`, f.users)
		testdb.Exec(t, pool, `DELETE FROM offers WHERE id = $1`, f.offerID)
		testdb.Exec(t, pool, `DELETE FROM sellers WHERE id = $1`, sellerID)
		testdb.Exec(t, pool, `DELETE FROM users WHERE id = ANY($1::uuid[])`, f.users)
		testdb.Exec(t, pool, `UPDATE products SET sales_count = $2 WHERE id = $1`, productID, salesBefore)
	})
	return f
}

func (f *fixture) order(user string, qty int) (Order, bool, error) {
	return f.store.CreateOrder(context.Background(), user, "key-"+testdb.Tag(), []CartItem{{OfferID: f.offerID, Qty: qty}})
}

// count runs a count(*) query and returns the number.
func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) keysWithStatus(status string) int {
	return f.count(`SELECT count(*) FROM product_keys WHERE offer_id = $1 AND status = $2::key_status`, f.offerID, status)
}

// The main guarantee: 40 buyers check out the same offer at the same instant,
// and only 3 keys exist. Exactly 3 must succeed, each with a different key.
func TestNoKeySoldTwice(t *testing.T) {
	const keys, buyers = 3, 40
	f := newFixture(t, keys, buyers)

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]error, buyers)
	orders := make([]Order, buyers)
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // all goroutines wait here, then go at once
			orders[i], _, results[i] = f.order(f.users[i], 1)
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded, outOfStock := 0, 0
	for _, err := range results {
		var stock *OutOfStockError
		switch {
		case err == nil:
			succeeded++
		case errors.As(err, &stock):
			outOfStock++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if succeeded != keys || outOfStock != buyers-keys {
		t.Fatalf("want %d orders and %d out-of-stock, got %d and %d", keys, buyers-keys, succeeded, outOfStock)
	}
	if n := f.count(`SELECT count(DISTINCT order_item_id) FROM product_keys WHERE offer_id = $1 AND status = 'reserved'`, f.offerID); n != keys {
		t.Errorf("want each reserved key on its own order line, got %d distinct lines", n)
	}

	// The winners all pay at once; every one must get a different key.
	seen := map[string]bool{}
	var mu sync.Mutex
	for i, err := range results {
		if err != nil {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			paid, err := f.store.Pay(context.Background(), f.users[i], orders[i].ID, TestCardSuccess)
			if err != nil {
				t.Errorf("pay: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, k := range paid.Items[0].Keys {
				if seen[k] {
					t.Errorf("key %s handed out twice", k)
				}
				seen[k] = true
			}
		}(i)
	}
	wg.Wait()
	if len(seen) != keys || f.keysWithStatus("sold") != keys {
		t.Errorf("want %d different sold keys, got %d (sold in DB: %d)", keys, len(seen), f.keysWithStatus("sold"))
	}
}

// A double click (the same Idempotency-Key sent many times at once) makes one order.
func TestSameIdempotencyKeyMakesOneOrder(t *testing.T) {
	f := newFixture(t, 10, 1)
	const tries = 20
	ids := make([]string, tries)
	created := make([]bool, tries)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < tries; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			o, c, err := f.store.CreateOrder(context.Background(), f.users[0], "same-click", []CartItem{{OfferID: f.offerID, Qty: 2}})
			if err != nil {
				t.Errorf("create: %v", err)
				return
			}
			ids[i], created[i] = o.ID, c
		}(i)
	}
	close(start)
	wg.Wait()

	newOrders := 0
	for i := range ids {
		if ids[i] != ids[0] {
			t.Fatalf("got different orders: %s and %s", ids[0], ids[i])
		}
		if created[i] {
			newOrders++
		}
	}
	if newOrders != 1 {
		t.Errorf("want exactly 1 newly created order, got %d", newOrders)
	}
	if n := f.keysWithStatus("reserved"); n != 2 {
		t.Errorf("want 2 reserved keys, got %d", n)
	}
}

// Asking for more keys than exist fails as a whole: no order, no reserved keys.
func TestShortOrderRollsBack(t *testing.T) {
	f := newFixture(t, 2, 1)
	_, _, err := f.order(f.users[0], 3)
	var stock *OutOfStockError
	if !errors.As(err, &stock) || stock.Available != 2 {
		t.Fatalf("want OutOfStockError{Available: 2}, got %v", err)
	}
	if n := f.count(`SELECT count(*) FROM orders WHERE user_id = $1`, f.users[0]); n != 0 {
		t.Errorf("order should have been rolled back, found %d", n)
	}
	if n := f.keysWithStatus("available"); n != 2 {
		t.Errorf("keys should be untouched, %d available", n)
	}
}

// After 15 minutes another buyer can take the keys, and the late payment is refused.
func TestExpiredReservation(t *testing.T) {
	f := newFixture(t, 1, 2)
	a, b := f.users[0], f.users[1]
	ctx := context.Background()

	orderA, _, err := f.order(a, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.order(b, 1); err == nil {
		t.Fatal("B should be out of stock while A holds the only key")
	}

	// Move A's reservation 16 minutes into the past.
	testdb.Exec(t, f.pool, `UPDATE orders SET created_at = now() - interval '16 minutes' WHERE id = $1`, orderA.ID)
	testdb.Exec(t, f.pool, `UPDATE product_keys SET reserved_until = now() - interval '1 minute' WHERE offer_id = $1`, f.offerID)

	orderB, _, err := f.order(b, 1)
	if err != nil {
		t.Fatalf("B should get the expired key: %v", err)
	}
	if _, err := f.store.Pay(ctx, a, orderA.ID, TestCardSuccess); !errors.Is(err, ErrReservationExpired) && !errors.Is(err, ErrNotPending) {
		t.Errorf("A's late payment: want reservation expired, got %v", err)
	}
	paid, err := f.store.Pay(ctx, b, orderB.ID, TestCardSuccess)
	if err != nil || len(paid.Items[0].Keys) != 1 {
		t.Fatalf("B should pay and get the key: %v", err)
	}
	if got, _ := f.store.Get(ctx, a, orderA.ID); got.Status != "expired" {
		t.Errorf("A's order: want expired, got %s", got.Status)
	}
}

// A declined card puts the keys straight back on sale.
func TestDeclineReleasesKeys(t *testing.T) {
	f := newFixture(t, 2, 1)
	ctx := context.Background()
	o, _, err := f.order(f.users[0], 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Pay(ctx, f.users[0], o.ID, "1234"); !errors.Is(err, ErrInvalidCard) {
		t.Errorf("invalid card: got %v", err)
	}
	if n := f.keysWithStatus("reserved"); n != 2 {
		t.Errorf("an invalid card must not release the keys, %d reserved", n)
	}
	if _, err := f.store.Pay(ctx, f.users[0], o.ID, TestCardDeclined); !errors.Is(err, ErrDeclined) {
		t.Fatalf("decline: got %v", err)
	}
	if n := f.keysWithStatus("available"); n != 2 {
		t.Errorf("want both keys available again, got %d", n)
	}
	if _, err := f.store.Pay(ctx, f.users[0], o.ID, TestCardSuccess); !errors.Is(err, ErrNotPending) {
		t.Errorf("paying a failed order: got %v", err)
	}
}

// Paying twice is harmless: one charge, same keys.
func TestPayTwice(t *testing.T) {
	f := newFixture(t, 1, 1)
	ctx := context.Background()
	o, _, err := f.order(f.users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	// The countdown comes from the database's clock: ~15 minutes for a new order.
	if o.SecondsLeft < 890 || o.SecondsLeft > 900 {
		t.Errorf("new order: want ~900 seconds left, got %d", o.SecondsLeft)
	}
	first, err := f.store.Pay(ctx, f.users[0], o.ID, TestCardSuccess)
	if err != nil {
		t.Fatal(err)
	}
	if first.SecondsLeft != 0 {
		t.Errorf("paid order: want 0 seconds left, got %d", first.SecondsLeft)
	}
	second, err := f.store.Pay(ctx, f.users[0], o.ID, TestCardSuccess)
	if err != nil || second.PaymentRef == nil || *second.PaymentRef != *first.PaymentRef {
		t.Errorf("second pay should return the same paid order: %v", err)
	}
}

// Nobody can read or pay someone else's order.
func TestOrdersArePrivate(t *testing.T) {
	f := newFixture(t, 1, 2)
	ctx := context.Background()
	o, _, err := f.order(f.users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Get(ctx, f.users[1], o.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user Get: got %v", err)
	}
	if _, err := f.store.Pay(ctx, f.users[1], o.ID, TestCardSuccess); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user Pay: got %v", err)
	}
}

// Starting a new checkout releases the keys of the user's previous unpaid order.
func TestNewCheckoutReplacesUnpaidOrder(t *testing.T) {
	f := newFixture(t, 2, 1)
	ctx := context.Background()
	first, _, err := f.order(f.users[0], 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.order(f.users[0], 2); err != nil {
		t.Fatalf("second checkout should reuse the same 2 keys: %v", err)
	}
	if got, _ := f.store.Get(ctx, f.users[0], first.ID); got.Status != "expired" {
		t.Errorf("first order: want expired, got %s", got.Status)
	}
}
