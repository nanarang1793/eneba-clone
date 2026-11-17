// Package orders turns a cart into an order with real keys.
//
// The flow:
//  1. POST /cart/quote   prices the cart from the database (never trusts the browser).
//  2. POST /orders       creates a pending order and RESERVES keys for 15 minutes.
//  3. POST /orders/{id}/pay  charges the (simulated) card. On success the
//     reserved keys become SOLD and the buyer can see them; on a decline
//     they're released back for sale.
//
// The guarantee that matters: one key is never given to two buyers. See
// reserveKeys for how a single SQL statement makes that true even when many
// people check out at the same moment.
package orders

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"eneba/backend/internal/catalog"
)

const (
	reservationMinutes = 15 // how long keys are held for an unpaid order
	maxLines           = 20 // different offers per cart
	maxQty             = 10 // keys of one offer per order
)

var (
	ErrNotFound           = errors.New("order not found")
	ErrNotPending         = errors.New("order is not awaiting payment")
	ErrReservationExpired = errors.New("reservation expired")
)

// OutOfStockError says one cart line can't be filled.
type OutOfStockError struct {
	OfferID   int64
	Available int
}

func (e *OutOfStockError) Error() string {
	if e.Available == 0 {
		return fmt.Sprintf("offer %d is sold out or no longer available", e.OfferID)
	}
	return fmt.Sprintf("only %d keys left for offer %d", e.Available, e.OfferID)
}

type Store struct {
	db       *pgxpool.Pool
	payments PaymentProvider
}

func NewStore(db *pgxpool.Pool, payments PaymentProvider) *Store {
	return &Store{db: db, payments: payments}
}

// ---------------------------------------------------------------- cart

// normalizeItems validates a cart and merges duplicate lines for the same offer.
func normalizeItems(items []CartItem) ([]CartItem, error) {
	if len(items) == 0 {
		return nil, errors.New("Your cart is empty")
	}
	var out []CartItem
	index := map[int64]int{} // offer_id -> position in out
	for _, it := range items {
		if it.OfferID <= 0 || it.Qty < 1 || it.Qty > maxQty {
			return nil, fmt.Errorf("Each item needs a valid offer and a quantity from 1 to %d", maxQty)
		}
		if i, seen := index[it.OfferID]; seen {
			out[i].Qty += it.Qty
			if out[i].Qty > maxQty {
				return nil, fmt.Errorf("You can buy at most %d keys of one offer", maxQty)
			}
			continue
		}
		index[it.OfferID] = len(out)
		out = append(out, it)
	}
	if len(out) > maxLines {
		return nil, fmt.Errorf("A cart can hold at most %d different items", maxLines)
	}
	return out, nil
}

// Quote prices a cart with today's prices and stock. Each line gets a
// status so the cart page can explain what changed.
func (s *Store) Quote(ctx context.Context, items []CartItem) (Quote, error) {
	q := Quote{Items: []QuoteLine{}, AllAvailable: true}
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.OfferID
	}

	type offerInfo struct {
		product    ProductRef
		seller     string
		priceCents int
		stock      int
	}
	found := map[int64]offerInfo{}
	rows, err := s.db.Query(ctx, `
        SELECT o.id, o.price_cents, s.name,
               p.slug, p.name, p.title, p.image_url, pl.name, r.name,
               (SELECT count(*) FROM product_keys k WHERE k.offer_id = o.id AND `+catalog.UsableKey+`)
        FROM offers o
        JOIN sellers s    ON s.id = o.seller_id
        JOIN products p   ON p.id = o.product_id
        JOIN platforms pl ON pl.id = p.platform_id
        JOIN regions r    ON r.id = p.region_id
        WHERE o.id = ANY($1) AND o.is_active`, ids)
	if err != nil {
		return q, fmt.Errorf("quote: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var o offerInfo
		if err := rows.Scan(&id, &o.priceCents, &o.seller, &o.product.Slug, &o.product.Name, &o.product.Title,
			&o.product.ImageURL, &o.product.Platform, &o.product.Region, &o.stock); err != nil {
			return q, fmt.Errorf("quote scan: %w", err)
		}
		found[id] = o
	}
	if err := rows.Err(); err != nil {
		return q, err
	}

	for _, it := range items {
		line := QuoteLine{OfferID: it.OfferID, Qty: it.Qty, Status: "unavailable"}
		if o, ok := found[it.OfferID]; ok {
			product := o.product
			line.Product = &product
			line.SellerName = o.seller
			line.PriceCents = o.priceCents
			line.Stock = o.stock
			line.LineTotalCents = o.priceCents * it.Qty
			switch {
			case o.stock == 0:
				line.Status = "sold_out"
			case o.stock < it.Qty:
				line.Status = "not_enough_stock"
			default:
				line.Status = "ok"
				q.TotalCents += line.LineTotalCents
			}
		}
		if line.Status != "ok" {
			q.AllAvailable = false
		}
		q.Items = append(q.Items, line)
	}
	return q, nil
}

// ---------------------------------------------------------------- create

// CreateOrder turns a cart into a pending order and reserves the keys, all in
// one transaction: either every line gets its keys, or nothing is saved.
//
// idemKey makes it safe to retry: the browser sends the same Idempotency-Key
// if the user double-clicks or the network drops, and the second request
// gets the first order back instead of creating (and reserving) a second one.
// created is false when an existing order was returned.
func (s *Store) CreateOrder(ctx context.Context, userID, idemKey string, items []CartItem) (o Order, created bool, err error) {
	if existing, ok, err := s.orderIDByKey(ctx, userID, idemKey); err != nil || ok {
		if err != nil {
			return o, false, err
		}
		o, err = s.Get(ctx, userID, existing)
		return o, false, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return o, false, err
	}
	defer tx.Rollback(ctx) // does nothing if we already committed

	// 0. A user has at most one unpaid order. Starting a new checkout (e.g.
	// after going back to the cart) releases the keys the old one held, so
	// they don't block this order or other buyers for 15 minutes.
	if err := closePendingOrders(ctx, tx, userID); err != nil {
		return o, false, err
	}

	// 1. Current prices, straight from the database.
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.OfferID
	}
	type offer struct {
		price, productID, sellerID int64
	}
	offers := map[int64]offer{}
	rows, err := tx.Query(ctx, `
        SELECT id, price_cents, product_id, seller_id FROM offers
        WHERE id = ANY($1) AND is_active`, ids)
	if err != nil {
		return o, false, fmt.Errorf("load offers: %w", err)
	}
	for rows.Next() {
		var id int64
		var of offer
		if err := rows.Scan(&id, &of.price, &of.productID, &of.sellerID); err != nil {
			rows.Close()
			return o, false, err
		}
		offers[id] = of
	}
	rows.Close()
	total := 0
	for _, it := range items {
		of, ok := offers[it.OfferID]
		if !ok {
			return o, false, &OutOfStockError{OfferID: it.OfferID}
		}
		total += int(of.price) * it.Qty
	}

	// 2. The order row.
	var orderID string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
        INSERT INTO orders (user_id, total_cents, idempotency_key)
        VALUES ($1, $2, $3)
        RETURNING id, created_at`, userID, total, idemKey,
	).Scan(&orderID, &createdAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		// The same request arrived twice at the same moment and the other one
		// won. Give back its order.
		tx.Rollback(ctx)
		id, _, err := s.orderIDByKey(ctx, userID, idemKey)
		if err != nil {
			return o, false, err
		}
		o, err = s.Get(ctx, userID, id)
		return o, false, err
	}
	if err != nil {
		return o, false, fmt.Errorf("insert order: %w", err)
	}
	reservedUntil := createdAt.Add(reservationMinutes * time.Minute)

	// 3. One order_items row per line (price copied now), then its keys.
	for _, it := range items {
		of := offers[it.OfferID]
		var itemID int64
		err := tx.QueryRow(ctx, `
            INSERT INTO order_items (order_id, offer_id, product_id, seller_id, unit_price_cents, quantity)
            VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			orderID, it.OfferID, of.productID, of.sellerID, of.price, it.Qty,
		).Scan(&itemID)
		if err != nil {
			return o, false, fmt.Errorf("insert order item: %w", err)
		}
		got, err := reserveKeys(ctx, tx, it.OfferID, itemID, it.Qty, reservedUntil)
		if err != nil {
			return o, false, err
		}
		if got < it.Qty {
			// Not enough keys: returning makes the deferred Rollback undo
			// the order AND every key reserved so far.
			return o, false, &OutOfStockError{OfferID: it.OfferID, Available: got}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return o, false, fmt.Errorf("commit order: %w", err)
	}
	o, err = s.Get(ctx, userID, orderID)
	return o, true, err
}

// reserveKeys marks up to qty usable keys of an offer as reserved for one
// order line and returns how many it got.
//
// Why two buyers can never get the same key:
//   - FOR UPDATE locks the rows the inner SELECT picks, until our
//     transaction ends. Nobody else can change them meanwhile.
//   - SKIP LOCKED makes other buyers skip rows that are already locked and
//     take the next free keys instead of waiting for us. So checkouts
//     running at the same moment each lock different keys, and nobody queues.
//   - The outer UPDATE then flips exactly those locked rows to 'reserved'.
//
// If fewer than qty rows come back, there weren't enough free keys.
func reserveKeys(ctx context.Context, tx pgx.Tx, offerID, itemID int64, qty int, until time.Time) (int, error) {
	tag, err := tx.Exec(ctx, `
        UPDATE product_keys
        SET status = 'reserved', reserved_until = $3, order_item_id = $2
        WHERE id IN (
            SELECT k.id FROM product_keys k
            WHERE k.offer_id = $1 AND `+catalog.UsableKey+`
            ORDER BY k.id
            LIMIT $4
            FOR UPDATE SKIP LOCKED
        )`, offerID, itemID, until, qty)
	if err != nil {
		return 0, fmt.Errorf("reserve keys: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (s *Store) orderIDByKey(ctx context.Context, userID, idemKey string) (string, bool, error) {
	var id string
	err := s.db.QueryRow(ctx, `SELECT id FROM orders WHERE user_id = $1 AND idempotency_key = $2`,
		userID, idemKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find order by key: %w", err)
	}
	return id, true, nil
}

// ---------------------------------------------------------------- pay

// Pay charges the card for a pending order, in one transaction:
//   - lock the order row, so two Pay clicks can't both charge;
//   - check this order still holds all its keys (reservation not expired);
//   - charge; on success the keys become 'sold', on a decline they're released.
func (s *Store) Pay(ctx context.Context, userID, orderID, card string) (Order, error) {
	if !isUUID(orderID) {
		return Order{}, ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	var total int
	err = tx.QueryRow(ctx, `
        SELECT status::text, total_cents FROM orders
        WHERE id = $1 AND user_id = $2
        FOR UPDATE`, orderID, userID).Scan(&status, &total)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound // also for other people's orders
	}
	if err != nil {
		return Order{}, fmt.Errorf("load order: %w", err)
	}
	if status == "paid" {
		tx.Rollback(ctx)
		return s.Get(ctx, userID, orderID) // paying twice is harmless
	}
	if status != "pending" {
		return Order{}, ErrNotPending
	}

	// Lock this order's keys that are still validly reserved. Once locked,
	// a checkout running at the same time can't take them even if the
	// reservation runs out during the charge (SKIP LOCKED skips them).
	var held, needed int
	err = tx.QueryRow(ctx, `
        SELECT
          (SELECT count(*) FROM (
              SELECT k.id FROM product_keys k
              JOIN order_items oi ON oi.id = k.order_item_id
              WHERE oi.order_id = $1 AND k.status = 'reserved' AND k.reserved_until > now()
              FOR UPDATE OF k) held),
          (SELECT coalesce(sum(quantity), 0) FROM order_items WHERE order_id = $1)`,
		orderID).Scan(&held, &needed)
	if err != nil {
		return Order{}, fmt.Errorf("check reservation: %w", err)
	}
	if held < needed {
		if err := closeOrder(ctx, tx, orderID, "expired"); err != nil {
			return Order{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, err
		}
		return Order{}, ErrReservationExpired
	}

	// A real provider would get its own idempotency key here too.
	ref, err := s.payments.Charge(ctx, total, card)
	if errors.Is(err, ErrDeclined) {
		if err := closeOrder(ctx, tx, orderID, "failed"); err != nil {
			return Order{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, err
		}
		return Order{}, ErrDeclined
	}
	if err != nil {
		return Order{}, err // ErrInvalidCard: nothing changed, the order stays pending
	}

	// Paid: hand over the keys and bump the popularity counters.
	steps := []string{
		`UPDATE product_keys SET status = 'sold', reserved_until = NULL
         WHERE status = 'reserved' AND order_item_id IN (SELECT id FROM order_items WHERE order_id = $1)`,
		`UPDATE products p SET sales_count = p.sales_count + x.n
         FROM (SELECT product_id, sum(quantity) n FROM order_items WHERE order_id = $1 GROUP BY product_id) x
         WHERE p.id = x.product_id`,
		`UPDATE sellers s SET sales_count = s.sales_count + x.n
         FROM (SELECT seller_id, sum(quantity) n FROM order_items WHERE order_id = $1 GROUP BY seller_id) x
         WHERE s.id = x.seller_id`,
	}
	for _, sql := range steps {
		if _, err := tx.Exec(ctx, sql, orderID); err != nil {
			return Order{}, fmt.Errorf("complete order: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid', paid_at = now(), payment_ref = $2 WHERE id = $1`,
		orderID, ref); err != nil {
		return Order{}, fmt.Errorf("mark paid: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit payment: %w", err)
	}
	return s.Get(ctx, userID, orderID)
}

// closeOrder marks an unpaid order failed/expired and puts its keys back on sale.
func closeOrder(ctx context.Context, tx pgx.Tx, orderID, status string) error {
	_, err := tx.Exec(ctx, `
        UPDATE product_keys SET status = 'available', reserved_until = NULL, order_item_id = NULL
        WHERE status = 'reserved' AND order_item_id IN (SELECT id FROM order_items WHERE order_id = $1)`, orderID)
	if err != nil {
		return fmt.Errorf("release keys: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE orders SET status = $2::order_status WHERE id = $1`, orderID, status)
	if err != nil {
		return fmt.Errorf("close order: %w", err)
	}
	return nil
}

// closePendingOrders expires all of a user's unpaid orders and releases their keys.
func closePendingOrders(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `
        UPDATE product_keys SET status = 'available', reserved_until = NULL, order_item_id = NULL
        WHERE status = 'reserved' AND order_item_id IN (
            SELECT oi.id FROM order_items oi JOIN orders o ON o.id = oi.order_id
            WHERE o.user_id = $1 AND o.status = 'pending')`, userID)
	if err != nil {
		return fmt.Errorf("release old reservations: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE orders SET status = 'expired' WHERE user_id = $1 AND status = 'pending'`, userID)
	if err != nil {
		return fmt.Errorf("expire old orders: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------- read

// expireOldOrders marks the user's unpaid orders past their reservation as
// expired. Done when orders are read instead of by a background job: there's
// no job to forget to run, and the keys themselves already count as free once
// reserved_until has passed (catalog.UsableKey).
func (s *Store) expireOldOrders(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, fmt.Sprintf(`
        UPDATE orders SET status = 'expired'
        WHERE user_id = $1 AND status = 'pending' AND created_at < now() - interval '%d minutes'`,
		reservationMinutes), userID)
	if err != nil {
		return fmt.Errorf("expire orders: %w", err)
	}
	return nil
}

// Get returns one of the user's orders, with keys if it's paid.
// Someone else's order is reported as not found, not as forbidden, so
// order ids can't be probed.
func (s *Store) Get(ctx context.Context, userID, orderID string) (Order, error) {
	if !isUUID(orderID) {
		return Order{}, ErrNotFound
	}
	if err := s.expireOldOrders(ctx, userID); err != nil {
		return Order{}, err
	}
	orders, err := s.load(ctx, `WHERE o.id = $1 AND o.user_id = $2`, orderID, userID)
	if err != nil {
		return Order{}, err
	}
	if len(orders) == 0 {
		return Order{}, ErrNotFound
	}
	o := orders[0]
	if o.Status == "paid" {
		if err := s.attachKeys(ctx, &o); err != nil {
			return o, err
		}
	}
	return o, nil
}

// List returns the user's 50 most recent orders, newest first (without keys).
func (s *Store) List(ctx context.Context, userID string) ([]Order, error) {
	if err := s.expireOldOrders(ctx, userID); err != nil {
		return nil, err
	}
	return s.load(ctx, `WHERE o.user_id = $1 ORDER BY o.created_at DESC LIMIT 50`, userID)
}

// load reads orders matching the given WHERE/ORDER clause, then their items.
func (s *Store) load(ctx context.Context, where string, args ...any) ([]Order, error) {
	rows, err := s.db.Query(ctx, `
        SELECT o.id, o.status::text, o.total_cents, o.currency, o.created_at, o.paid_at, o.payment_ref
        FROM orders o `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("load orders: %w", err)
	}
	orders := []Order{}
	byID := map[string]int{}
	ids := []string{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.Status, &o.TotalCents, &o.Currency, &o.CreatedAt, &o.PaidAt, &o.PaymentRef); err != nil {
			rows.Close()
			return nil, err
		}
		o.ExpiresAt = o.CreatedAt.Add(reservationMinutes * time.Minute)
		o.Items = []OrderItem{}
		byID[o.ID] = len(orders)
		ids = append(ids, o.ID)
		orders = append(orders, o)
	}
	rows.Close()
	if len(ids) == 0 {
		return orders, rows.Err()
	}

	rows, err = s.db.Query(ctx, `
        SELECT oi.order_id, oi.id, oi.offer_id, p.slug, p.name, p.title, p.image_url, pl.name, r.name,
               s.name, oi.unit_price_cents, oi.quantity
        FROM order_items oi
        JOIN products p   ON p.id = oi.product_id
        JOIN platforms pl ON pl.id = p.platform_id
        JOIN regions r    ON r.id = p.region_id
        JOIN sellers s    ON s.id = oi.seller_id
        WHERE oi.order_id = ANY($1::uuid[])
        ORDER BY oi.id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var orderID string
		var it OrderItem
		if err := rows.Scan(&orderID, &it.ID, &it.OfferID, &it.Product.Slug, &it.Product.Name, &it.Product.Title,
			&it.Product.ImageURL, &it.Product.Platform, &it.Product.Region,
			&it.SellerName, &it.UnitPriceCents, &it.Quantity); err != nil {
			return nil, err
		}
		it.Keys = []string{}
		o := &orders[byID[orderID]]
		o.Items = append(o.Items, it)
	}
	return orders, rows.Err()
}

// attachKeys fills in the sold key codes of a paid order.
func (s *Store) attachKeys(ctx context.Context, o *Order) error {
	rows, err := s.db.Query(ctx, `
        SELECT k.order_item_id, k.code FROM product_keys k
        JOIN order_items oi ON oi.id = k.order_item_id
        WHERE oi.order_id = $1 AND k.status = 'sold'
        ORDER BY k.id`, o.ID)
	if err != nil {
		return fmt.Errorf("load keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var itemID int64
		var code string
		if err := rows.Scan(&itemID, &code); err != nil {
			return err
		}
		for i := range o.Items {
			if o.Items[i].ID == itemID {
				o.Items[i].Keys = append(o.Items[i].Keys, code)
			}
		}
	}
	return rows.Err()
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID rejects malformed ids before they reach Postgres (which would
// answer with an error instead of "not found").
func isUUID(s string) bool { return uuidPattern.MatchString(s) }
