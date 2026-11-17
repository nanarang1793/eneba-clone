// Command seedgen writes deterministic seed data as SQL to stdout.
//
//	go run ./cmd/seedgen > seed/seed.sql
//
// The output is plain SQL so it can be loaded with psql alone (no Go or DB
// driver needed in the seed container). A fixed random seed makes the
// output identical on every run, so seed.sql diffs stay meaningful.
package main

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	randSeed       = 2027
	targetProducts = 200
)

type platform struct {
	slug, name string
	device     string // shown in brackets in the product name, e.g. "PC", "PS5"
	store      string // key store, e.g. "Steam", "PSN"
}

type region struct {
	slug, name, label string
	priceMult         float64 // regional pricing differences
}

var platforms = []platform{
	{"steam", "Steam", "PC", "Steam"},
	{"ea-app", "EA App", "PC", "EA App"},
	{"ubisoft-connect", "Ubisoft Connect", "PC", "Ubisoft Connect"},
	{"battle-net", "Battle.net", "PC", "Battle.net"},
	{"xbox", "Xbox", "Xbox Series X|S", "Xbox Live"},
	{"psn", "PlayStation", "PS5", "PSN"},
	{"nintendo", "Nintendo", "Nintendo Switch", "Nintendo eShop"},
}

var regions = []region{
	{"global", "Global", "GLOBAL", 1.00},
	{"europe", "Europe", "EUROPE", 0.97},
	{"north-america", "North America", "NORTH AMERICA", 0.93},
	{"united-kingdom", "United Kingdom", "UNITED KINGDOM", 0.95},
	{"latin-america", "Latin America", "LATAM", 0.78},
}

type seller struct {
	slug, name string
	rating     float64
	sales      int
}

var sellers = []seller{
	{"pixelvault", "PixelVault", 4.91, 184_220},
	{"keyhaven", "KeyHaven", 4.84, 96_410},
	{"baltikeys", "BaltiKeys", 4.97, 312_905},
	{"nordkeys", "NordKeys", 4.62, 41_870},
	{"lootlane", "LootLane", 4.78, 73_115},
	{"codecrate", "CodeCrate", 4.55, 18_342},
	{"gamedrop", "GameDrop", 4.88, 150_067},
	{"instaplay", "InstaPlay", 4.36, 9_480},
}

type title struct {
	name, publisher, genre string
	year                   int
	msrp                   float64 // EUR
	platforms              []string
	kind                   string // game | dlc
}

// Game titles, most popular first (list order drives the seeded sales_count).
var titles = []title{
	{"Elden Ring", "Bandai Namco", "action RPG", 2022, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Cyberpunk 2077", "CD PROJEKT RED", "open-world RPG", 2020, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Baldur's Gate 3", "Larian Studios", "RPG", 2023, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Red Dead Redemption 2", "Rockstar Games", "open-world action", 2019, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Grand Theft Auto V", "Rockstar Games", "open-world action", 2015, 29.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Minecraft: Java & Bedrock Edition", "Mojang Studios", "sandbox", 2011, 29.99, []string{"xbox", "nintendo"}, "game"},
	{"EA SPORTS FC 25", "Electronic Arts", "sports", 2024, 69.99, []string{"ea-app", "xbox", "psn"}, "game"},
	{"Hogwarts Legacy", "Warner Bros. Games", "action RPG", 2023, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"The Witcher 3: Wild Hunt Complete Edition", "CD PROJEKT RED", "open-world RPG", 2015, 49.99, []string{"steam", "xbox", "psn", "nintendo"}, "game"},
	{"Call of Duty: Black Ops 6", "Activision", "shooter", 2024, 69.99, []string{"battle-net", "xbox", "psn"}, "game"},
	{"Helldivers 2", "Sony Interactive Entertainment", "co-op shooter", 2024, 39.99, []string{"steam", "psn"}, "game"},
	{"Stardew Valley", "ConcernedApe", "farming sim", 2016, 13.99, []string{"steam", "xbox", "nintendo"}, "game"},
	{"Hades II", "Supergiant Games", "roguelike", 2025, 29.99, []string{"steam", "nintendo"}, "game"},
	{"Black Myth: Wukong", "Game Science", "action RPG", 2024, 59.99, []string{"steam", "psn"}, "game"},
	{"Forza Horizon 5", "Xbox Game Studios", "racing", 2021, 59.99, []string{"steam", "xbox"}, "game"},
	{"Diablo IV", "Blizzard Entertainment", "action RPG", 2023, 69.99, []string{"battle-net", "xbox", "psn"}, "game"},
	{"Starfield", "Bethesda Softworks", "space RPG", 2023, 69.99, []string{"steam", "xbox"}, "game"},
	{"The Legend of Zelda: Tears of the Kingdom", "Nintendo", "action adventure", 2023, 69.99, []string{"nintendo"}, "game"},
	{"Mario Kart 8 Deluxe", "Nintendo", "racing", 2017, 59.99, []string{"nintendo"}, "game"},
	{"Super Smash Bros. Ultimate", "Nintendo", "fighting", 2018, 59.99, []string{"nintendo"}, "game"},
	{"Animal Crossing: New Horizons", "Nintendo", "life sim", 2020, 59.99, []string{"nintendo"}, "game"},
	{"Assassin's Creed Shadows", "Ubisoft", "action adventure", 2025, 69.99, []string{"ubisoft-connect", "xbox", "psn"}, "game"},
	{"Assassin's Creed Valhalla", "Ubisoft", "action adventure", 2020, 59.99, []string{"ubisoft-connect", "xbox", "psn"}, "game"},
	{"Far Cry 6", "Ubisoft", "shooter", 2021, 59.99, []string{"ubisoft-connect", "xbox", "psn"}, "game"},
	{"Tom Clancy's Rainbow Six Siege", "Ubisoft", "tactical shooter", 2015, 19.99, []string{"ubisoft-connect", "xbox", "psn"}, "game"},
	{"Battlefield 2042", "Electronic Arts", "shooter", 2021, 59.99, []string{"ea-app", "xbox", "psn"}, "game"},
	{"The Sims 4 Deluxe Party Edition", "Electronic Arts", "life sim", 2014, 39.99, []string{"ea-app", "xbox", "psn"}, "game"},
	{"Need for Speed Unbound", "Electronic Arts", "racing", 2022, 69.99, []string{"ea-app", "xbox", "psn"}, "game"},
	{"Overwatch 2 Watchpoint Pack", "Blizzard Entertainment", "hero shooter", 2022, 39.99, []string{"battle-net", "xbox", "psn"}, "game"},
	{"Sekiro: Shadows Die Twice", "Activision", "action adventure", 2019, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Dark Souls III", "Bandai Namco", "action RPG", 2016, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Resident Evil 4", "Capcom", "survival horror", 2023, 39.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Monster Hunter Wilds", "Capcom", "action RPG", 2025, 69.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Street Fighter 6", "Capcom", "fighting", 2023, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Tekken 8", "Bandai Namco", "fighting", 2024, 69.99, []string{"steam", "xbox", "psn"}, "game"},
	{"It Takes Two", "Electronic Arts", "co-op adventure", 2021, 39.99, []string{"ea-app", "xbox", "psn", "nintendo"}, "game"},
	{"Hollow Knight", "Team Cherry", "metroidvania", 2017, 14.99, []string{"steam", "xbox", "nintendo"}, "game"},
	{"Hollow Knight: Silksong", "Team Cherry", "metroidvania", 2025, 19.99, []string{"steam", "xbox", "psn", "nintendo"}, "game"},
	{"Terraria", "Re-Logic", "sandbox", 2011, 9.99, []string{"steam", "xbox", "nintendo"}, "game"},
	{"Palworld", "Pocketpair", "survival", 2024, 29.99, []string{"steam", "xbox"}, "game"},
	{"Lies of P", "Neowiz", "action RPG", 2023, 59.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Doom Eternal", "Bethesda Softworks", "shooter", 2020, 39.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Fallout 4", "Bethesda Softworks", "open-world RPG", 2015, 19.99, []string{"steam", "xbox", "psn"}, "game"},
	{"The Elder Scrolls V: Skyrim Anniversary Edition", "Bethesda Softworks", "open-world RPG", 2021, 49.99, []string{"steam", "xbox", "psn", "nintendo"}, "game"},
	{"Sid Meier's Civilization VII", "2K", "strategy", 2025, 69.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Cities: Skylines II", "Paradox Interactive", "city builder", 2023, 49.99, []string{"steam", "xbox"}, "game"},
	{"Total War: WARHAMMER III", "SEGA", "strategy", 2022, 59.99, []string{"steam"}, "game"},
	{"Satisfactory", "Coffee Stain Publishing", "factory sim", 2024, 38.99, []string{"steam"}, "game"},
	{"Sea of Thieves", "Xbox Game Studios", "co-op adventure", 2018, 39.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Ghost of Tsushima Director's Cut", "Sony Interactive Entertainment", "action adventure", 2024, 59.99, []string{"steam", "psn"}, "game"},
	{"Marvel's Spider-Man Remastered", "Sony Interactive Entertainment", "action adventure", 2022, 59.99, []string{"steam", "psn"}, "game"},
	{"God of War Ragnarök", "Sony Interactive Entertainment", "action adventure", 2024, 59.99, []string{"steam", "psn"}, "game"},
	{"Death Stranding Director's Cut", "505 Games", "action adventure", 2022, 39.99, []string{"steam", "psn"}, "game"},
	{"Hitman World of Assassination", "IO Interactive", "stealth", 2023, 69.99, []string{"steam", "xbox", "psn"}, "game"},
	{"Rust", "Facepunch Studios", "survival", 2018, 39.99, []string{"steam"}, "game"},
	{"Valheim", "Coffee Stain Publishing", "survival", 2021, 19.99, []string{"steam", "xbox"}, "game"},
	{"Lethal Company", "Zeekerss", "co-op horror", 2023, 9.75, []string{"steam"}, "game"},
	{"Elden Ring Shadow of the Erdtree", "Bandai Namco", "action RPG", 2024, 39.99, []string{"steam", "xbox", "psn"}, "dlc"},
	{"Cyberpunk 2077: Phantom Liberty", "CD PROJEKT RED", "open-world RPG", 2023, 29.99, []string{"steam", "xbox", "psn"}, "dlc"},
	{"The Sims 4 Cottage Living", "Electronic Arts", "life sim", 2021, 39.99, []string{"ea-app"}, "dlc"},
}

type card struct {
	platform, region, name string
	face                   float64 // face value in EUR-equivalent
	kind                   string  // gift_card | subscription
	publisher              string
}

var cards = []card{
	{"steam", "europe", "Steam Gift Card 10 EUR", 10, "gift_card", "Valve"},
	{"steam", "europe", "Steam Gift Card 20 EUR", 20, "gift_card", "Valve"},
	{"steam", "europe", "Steam Gift Card 50 EUR", 50, "gift_card", "Valve"},
	{"steam", "north-america", "Steam Gift Card 20 USD", 18.5, "gift_card", "Valve"},
	{"steam", "north-america", "Steam Gift Card 50 USD", 46, "gift_card", "Valve"},
	{"psn", "europe", "PlayStation Store Gift Card 10 EUR", 10, "gift_card", "Sony Interactive Entertainment"},
	{"psn", "europe", "PlayStation Store Gift Card 20 EUR", 20, "gift_card", "Sony Interactive Entertainment"},
	{"psn", "europe", "PlayStation Store Gift Card 50 EUR", 50, "gift_card", "Sony Interactive Entertainment"},
	{"psn", "united-kingdom", "PlayStation Store Gift Card 20 GBP", 23.5, "gift_card", "Sony Interactive Entertainment"},
	{"xbox", "europe", "Xbox Gift Card 15 EUR", 15, "gift_card", "Microsoft"},
	{"xbox", "europe", "Xbox Gift Card 25 EUR", 25, "gift_card", "Microsoft"},
	{"xbox", "north-america", "Xbox Gift Card 25 USD", 23, "gift_card", "Microsoft"},
	{"nintendo", "europe", "Nintendo eShop Card 15 EUR", 15, "gift_card", "Nintendo"},
	{"nintendo", "europe", "Nintendo eShop Card 25 EUR", 25, "gift_card", "Nintendo"},
	{"nintendo", "europe", "Nintendo eShop Card 50 EUR", 50, "gift_card", "Nintendo"},
	{"battle-net", "europe", "Battle.net Gift Card 20 EUR", 20, "gift_card", "Blizzard Entertainment"},
	{"xbox", "europe", "Xbox Game Pass Ultimate 1 Month", 17.99, "subscription", "Microsoft"},
	{"xbox", "europe", "Xbox Game Pass Ultimate 3 Months", 53.97, "subscription", "Microsoft"},
	{"xbox", "global", "PC Game Pass 3 Months", 35.97, "subscription", "Microsoft"},
	{"psn", "europe", "PlayStation Plus Essential 12 Months", 71.99, "subscription", "Sony Interactive Entertainment"},
	{"psn", "united-kingdom", "PlayStation Plus Extra 12 Months", 117.99, "subscription", "Sony Interactive Entertainment"},
	{"nintendo", "europe", "Nintendo Switch Online 12 Months Family Membership", 34.99, "subscription", "Nintendo"},
	{"ea-app", "global", "EA Play 1 Month", 5.99, "subscription", "Electronic Arts"},
}

type product struct {
	id                int
	slug, name, title string
	description       string
	kind              string
	platformID        int
	regionID          int
	msrpCents         int
	releaseDate       string
	publisher         string
	salesCount        int
}

type offer struct {
	id, productID, sellerID int
	priceCents              int
	active                  bool
	stock                   int
}

func main() {
	rng := rand.New(rand.NewSource(randSeed))
	platformIdx := map[string]int{}
	for i, p := range platforms {
		platformIdx[p.slug] = i
	}
	regionIdx := map[string]int{}
	for i, r := range regions {
		regionIdx[r.slug] = i
	}

	// ---- build candidate products -------------------------------------
	var all []product
	popularity := func(rank int) int { return int(5000.0 / (1 + float64(rank)/4)) }

	for rank, t := range titles {
		for _, ps := range t.platforms {
			p := platforms[platformIdx[ps]]
			// Every title gets GLOBAL or EUROPE; some also get an extra region.
			regs := []string{"global"}
			if p.slug == "psn" || p.slug == "nintendo" || rng.Float64() < 0.35 {
				regs = []string{"europe"}
			}
			if rng.Float64() < 0.45 {
				extra := []string{"north-america", "united-kingdom", "latin-america"}[rng.Intn(3)]
				regs = append(regs, extra)
			}
			for _, rs := range regs {
				r := regions[regionIdx[rs]]
				name := fmt.Sprintf("%s (%s) %s Key %s", t.name, p.device, p.store, r.label)
				if t.kind == "dlc" {
					name = fmt.Sprintf("%s (DLC) (%s) %s Key %s", t.name, p.device, p.store, r.label)
				}
				all = append(all, product{
					name:        name,
					title:       t.name,
					description: describe(t.name, t.genre, t.publisher, t.kind, p, r),
					kind:        t.kind,
					platformID:  platformIdx[ps] + 1,
					regionID:    regionIdx[rs] + 1,
					msrpCents:   toCents(t.msrp * r.priceMult),
					releaseDate: fmt.Sprintf("%d-%02d-%02d", t.year, 1+rng.Intn(12), 1+rng.Intn(28)),
					publisher:   t.publisher,
					salesCount:  jitter(rng, popularity(rank)),
				})
			}
		}
	}

	// Gift cards and subscriptions are always included.
	var fixed []product
	for i, c := range cards {
		p := platforms[platformIdx[c.platform]]
		r := regions[regionIdx[c.region]]
		fixed = append(fixed, product{
			name:        fmt.Sprintf("%s %s Key %s", c.name, p.store, r.label),
			title:       c.name,
			description: describeCard(c, p, r),
			kind:        c.kind,
			platformID:  platformIdx[c.platform] + 1,
			regionID:    regionIdx[c.region] + 1,
			msrpCents:   toCents(c.face),
			releaseDate: fmt.Sprintf("2024-%02d-01", 1+i%12),
			publisher:   c.publisher,
			salesCount:  jitter(rng, 2500),
		})
	}

	// Pick a deterministic subset of games so the total is targetProducts.
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	need := targetProducts - len(fixed)
	if need > len(all) {
		need = len(all)
	}
	products := append(all[:need], fixed...)
	sort.SliceStable(products, func(i, j int) bool { return products[i].name < products[j].name })

	seenSlug := map[string]bool{}
	for i := range products {
		products[i].id = i + 1
		s := slugify(products[i].name)
		for seenSlug[s] {
			s += "-x"
		}
		seenSlug[s] = true
		products[i].slug = s
	}

	// ---- offers --------------------------------------------------------
	var offers []offer
	for _, p := range products {
		n := 1 + rng.Intn(6) // 1..6 sellers
		if rng.Float64() < 0.15 {
			n = 7 + rng.Intn(2) // a few very competitive products
		}
		if n > len(sellers) {
			n = len(sellers)
		}
		for _, si := range rng.Perm(len(sellers))[:n] {
			var factor float64
			switch p.kind {
			case "gift_card":
				factor = 1.00 + rng.Float64()*0.12 // cards sell around face value
			case "subscription":
				factor = 0.60 + rng.Float64()*0.35
			default:
				factor = 0.40 + rng.Float64()*0.55 // keys well below retail
			}
			stock := 1 + rng.Intn(12)
			if rng.Float64() < 0.08 {
				stock = 0 // sold out
			}
			offers = append(offers, offer{
				id:         len(offers) + 1,
				productID:  p.id,
				sellerID:   si + 1,
				priceCents: endIn9(int(math.Round(float64(p.msrpCents) * factor))),
				active:     rng.Float64() > 0.03,
				stock:      stock,
			})
		}
	}

	write(products, offers)
}

func write(products []product, offers []offer) {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	p := func(format string, a ...any) { fmt.Fprintf(w, format, a...) }

	p("-- Generated by backend/cmd/seedgen. Do not edit by hand; run `make seed-gen`.\n")
	p("-- %d products, %d sellers, %d offers.\n\n", len(products), len(sellers), len(offers))
	p("BEGIN;\n\n")
	p("TRUNCATE product_keys, order_items, orders, offers, products, sellers,\n")
	p("         platforms, regions, home_blocks, banners, sessions, users\n")
	p("         RESTART IDENTITY CASCADE;\n\n")

	p("INSERT INTO platforms (id, slug, name) VALUES\n")
	for i, pl := range platforms {
		p("  (%d, %s, %s)%s\n", i+1, q(pl.slug), q(pl.name), sep(i, len(platforms)))
	}
	p("\nINSERT INTO regions (id, slug, name) VALUES\n")
	for i, r := range regions {
		p("  (%d, %s, %s)%s\n", i+1, q(r.slug), q(r.name), sep(i, len(regions)))
	}

	p("\nINSERT INTO sellers (id, slug, name, rating, sales_count) VALUES\n")
	for i, s := range sellers {
		p("  (%d, %s, %s, %.2f, %d)%s\n", i+1, q(s.slug), q(s.name), s.rating, s.sales, sep(i, len(sellers)))
	}

	p("\nINSERT INTO products (id, slug, name, title, description, type, platform_id, region_id, msrp_cents, release_date, publisher, sales_count) VALUES\n")
	for i, pr := range products {
		p("  (%d, %s, %s, %s, %s, %s, %d, %d, %d, %s, %s, %d)%s\n",
			pr.id, q(pr.slug), q(pr.name), q(pr.title), q(pr.description), q(pr.kind),
			pr.platformID, pr.regionID, pr.msrpCents, q(pr.releaseDate), q(pr.publisher), pr.salesCount,
			sep(i, len(products)))
	}

	p("\nINSERT INTO offers (id, product_id, seller_id, price_cents, is_active) VALUES\n")
	for i, o := range offers {
		p("  (%d, %d, %d, %d, %t)%s\n", o.id, o.productID, o.sellerID, o.priceCents, o.active, sep(i, len(offers)))
	}

	// Keys are generated in SQL from (offer_id, stock) pairs to keep this
	// file small. Codes are derived from md5 so they're deterministic.
	p("\n-- Keys: one row per unit of stock, code like ABCDE-12345-FGHIJ.\n")
	p("INSERT INTO product_keys (offer_id, code)\n")
	p("SELECT s.offer_id,\n")
	p("       upper(substr(h, 1, 5) || '-' || substr(h, 6, 5) || '-' || substr(h, 11, 5))\n")
	p("FROM (VALUES\n")
	var withStock []offer
	for _, o := range offers {
		if o.stock > 0 {
			withStock = append(withStock, o)
		}
	}
	for i, o := range withStock {
		end := ","
		if i == len(withStock)-1 {
			end = ""
		}
		p("  (%d, %d)%s\n", o.id, o.stock, end)
	}
	p(") AS s(offer_id, stock)\n")
	p("CROSS JOIN LATERAL generate_series(1, s.stock) AS g(n)\n")
	p("CROSS JOIN LATERAL (SELECT md5('eneba-seed-' || s.offer_id || '-' || g.n) AS h) AS k;\n")

	w.WriteString(`
-- Homepage layout: the frontend renders these in order and fetches each
-- block's data from its own provider endpoint.
INSERT INTO home_blocks (type, title, position, config) VALUES
  ('banner',       'Featured',            1, '{}'),
  ('best_sellers', 'Bestsellers',         2, '{"limit": 12}'),
  ('deals',        'Hot deals',           3, '{"limit": 12, "min_discount_pct": 40}'),
  ('new_releases', 'New releases',        4, '{"limit": 12}'),
  ('by_type',      'Gift cards',          5, '{"limit": 12, "type": "gift_card"}'),
  ('by_type',      'Subscriptions',       6, '{"limit": 12, "type": "subscription"}');

INSERT INTO banners (title, subtitle, cta_label, cta_url, position) VALUES
  ('Top up and save',            'Gift cards for every platform, delivered instantly', 'Shop gift cards', '/store?type=gift_card', 1),
  ('Bestselling games',          'The games everyone is buying right now',             'Shop bestsellers', '/store?type=game&sort=popular', 2),
  ('New this season',            'The latest releases from verified sellers',          'Browse new',      '/store?sort=newest', 3);

-- Demo account: demo@eneba.test / demo1234 (bcrypt via pgcrypto).
INSERT INTO users (email, password_hash, display_name)
VALUES ('demo@eneba.test', crypt('demo1234', gen_salt('bf', 10)), 'Demo User');

-- Move sequences past the explicit ids so later inserts don't collide.
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['platforms', 'regions', 'sellers', 'products', 'offers'] LOOP
    EXECUTE format('SELECT setval(pg_get_serial_sequence(%L, ''id''), (SELECT max(id) FROM %I))', t, t);
  END LOOP;
END $$;

COMMIT;
`)
}

// ---- helpers -------------------------------------------------------------

func describe(name, genre, publisher, kind string, p platform, r region) string {
	what := "game"
	if kind == "dlc" {
		what = "expansion"
	}
	return fmt.Sprintf("%s is a %s %s from %s. This is a digital %s key for the %s region. "+
		"Redeem it on %s to add the %s to your library and download it instantly.",
		name, genre, what, publisher, p.store, r.name, p.name, what)
}

func describeCard(c card, p platform, r region) string {
	if c.kind == "subscription" {
		return fmt.Sprintf("%s for %s, %s region. Redeem the code on your %s account to start or extend your membership.",
			c.name, p.name, r.name, p.store)
	}
	return fmt.Sprintf("%s for the %s region. Redeem it on %s to top up your wallet and buy games, DLC and in-game items.",
		c.name, r.name, p.store)
}

func toCents(eur float64) int { return endIn9(int(math.Round(eur * 100))) }

// endIn9 makes prices look like store prices (…9 cents).
func endIn9(c int) int {
	if c < 10 {
		return 9
	}
	return c - c%10 + 9
}

func jitter(rng *rand.Rand, base int) int {
	return int(float64(base) * (0.5 + rng.Float64()))
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("'", "", "’", "", "&", "and", "|", "", "ö", "o").Replace(s)
	return strings.Trim(nonAlnum.ReplaceAllString(s, "-"), "-")
}

// q quotes a string as a SQL literal.
func q(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func sep(i, n int) string {
	if i == n-1 {
		return ";"
	}
	return ","
}
