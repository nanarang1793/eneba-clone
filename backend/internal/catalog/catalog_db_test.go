package catalog

// Integration tests against the seed data (TEST_DATABASE_URL; skipped without it).

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"eneba/backend/internal/testdb"
)

func list(t *testing.T, s *Store, query string) ListResult {
	t.Helper()
	v, _ := url.ParseQuery(query)
	p, err := ParseListParams(v)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	res, err := s.ListProducts(context.Background(), p)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return res
}

func TestSearchFindsTyposAndShortNames(t *testing.T) {
	s := NewStore(testdb.Open(t))
	cases := map[string]string{
		"q=eldn+ring":  "Elden Ring",            // typo
		"q=witchr":     "The Witcher 3",         // typo, partial word
		"q=gta":        "Grand Theft Auto V",    // synonym
		"q=rdr2":       "Red Dead Redemption 2", // synonym
		"q=CYBERPUNK":  "Cyberpunk 2077",        // case
		"q=hollow+kni": "Hollow Knight",         // prefix
	}
	for query, wantTitle := range cases {
		res := list(t, s, query)
		if res.Total == 0 || !strings.HasPrefix(res.Items[0].Title, wantTitle) {
			first := "(none)"
			if res.Total > 0 {
				first = res.Items[0].Title
			}
			t.Errorf("%s: want first result %q, got %s", query, wantTitle, first)
		}
	}
	if res := list(t, s, "q=zzzqqqxx"); res.Total != 0 {
		t.Errorf("nonsense query should find nothing, found %d", res.Total)
	}
	// LIKE wildcards in the search box are matched literally, not as wildcards.
	if res := list(t, s, "q=%25"); res.Total != 0 {
		t.Errorf("q=%% should find nothing, found %d", res.Total)
	}
}

func TestFiltersAndFacets(t *testing.T) {
	s := NewStore(testdb.Open(t))
	all := list(t, s, "")

	// Without filters, each facet's counts add up to the total.
	sum := func(fs []FacetCount) (n int) {
		for _, f := range fs {
			n += f.Count
		}
		return n
	}
	if sum(all.Facets.Platforms) != all.Total || sum(all.Facets.Regions) != all.Total || sum(all.Facets.Types) != all.Total {
		t.Errorf("facet sums should equal total %d", all.Total)
	}

	// Ticking Steam: total = Steam's count, and the other platforms still show
	// their counts (a facet ignores its own filter).
	steam := list(t, s, "platform=steam")
	for _, f := range steam.Facets.Platforms {
		for _, g := range all.Facets.Platforms {
			if f.Slug == g.Slug && f.Count != g.Count {
				t.Errorf("platform facet %s changed when filtering by platform: %d vs %d", f.Slug, f.Count, g.Count)
			}
		}
		if f.Slug == "steam" && f.Count != steam.Total {
			t.Errorf("steam facet %d != total %d", f.Count, steam.Total)
		}
	}
	for _, it := range steam.Items {
		if it.Platform.Slug != "steam" {
			t.Errorf("non-Steam item in Steam results: %s", it.Name)
		}
	}

	// Price filter compares the cheapest in-stock offer, in cents.
	for _, it := range list(t, s, "min_price=10&max_price=20&per_page=60").Items {
		if it.PriceCents < 1000 || it.PriceCents > 2000 {
			t.Errorf("%s costs %d, outside 10–20 €", it.Name, it.PriceCents)
		}
	}
}

// Walking every page sorted by price must show each product exactly once, in order.
func TestPaginationIsStableAndComplete(t *testing.T) {
	s := NewStore(testdb.Open(t))
	first := list(t, s, "sort=price_asc&per_page=25")
	seen := map[int64]bool{}
	last := 0
	for page := 1; page <= first.TotalPages; page++ {
		res := list(t, s, "sort=price_asc&per_page=25&page="+strconv.Itoa(page))
		for _, it := range res.Items {
			if seen[it.ID] {
				t.Fatalf("product %d appeared twice (page %d)", it.ID, page)
			}
			if it.PriceCents < last {
				t.Fatalf("not sorted: %d after %d", it.PriceCents, last)
			}
			seen[it.ID], last = true, it.PriceCents
		}
	}
	if len(seen) != first.Total {
		t.Errorf("saw %d products over all pages, total says %d", len(seen), first.Total)
	}
}
