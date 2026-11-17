package catalog

import (
	"net/url"
	"testing"
)

func TestParseListParamsDefaults(t *testing.T) {
	p, err := ParseListParams(url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Sort != SortRelevance || p.Page != 1 || p.PerPage != defaultPerPage {
		t.Errorf("defaults: got sort=%q page=%d per_page=%d", p.Sort, p.Page, p.PerPage)
	}
	if p.MinCents != nil || p.MaxCents != nil || p.Q != "" {
		t.Errorf("expected no filters, got %+v", p)
	}
}

func TestParseListParamsValues(t *testing.T) {
	v, _ := url.ParseQuery("q=+Elden+&platform=steam,XBOX&platform=psn&region=global&type=game&min_price=10&max_price=29.99&sort=price_asc&page=2&per_page=30")
	p, err := ParseListParams(v)
	if err != nil {
		t.Fatal(err)
	}
	if p.Q != "Elden" {
		t.Errorf("q should be trimmed, got %q", p.Q)
	}
	// Comma lists and repeated params both work; slugs are lowercased.
	if got := p.Platforms; len(got) != 3 || got[0] != "steam" || got[1] != "xbox" || got[2] != "psn" {
		t.Errorf("platforms: %v", got)
	}
	if len(p.Types) != 1 || p.Types[0] != "game" {
		t.Errorf("types: %v", p.Types)
	}
	// Euros in the URL, integer cents inside the backend.
	if *p.MinCents != 1000 || *p.MaxCents != 2999 {
		t.Errorf("prices: %d %d", *p.MinCents, *p.MaxCents)
	}
	if p.Page != 2 || p.PerPage != 30 || p.Sort != SortPriceAsc {
		t.Errorf("paging/sort: %+v", p)
	}
}

func TestParseListParamsRejectsBadInput(t *testing.T) {
	bad := []string{
		"sort=cheapest",
		"page=0",
		"page=abc",
		"per_page=61",
		"min_price=-1",
		"max_price=lots",
		"min_price=30&max_price=10",
		"q=" + string(make([]byte, 101)),
	}
	for _, raw := range bad {
		v, _ := url.ParseQuery(raw)
		if _, err := ParseListParams(v); err == nil {
			t.Errorf("%q: expected an error", raw)
		}
	}
}

func TestExpandQuery(t *testing.T) {
	cases := map[string]string{
		"gta":         "grand theft auto",
		"GTA 5":       "grand theft auto 5",
		"cod":         "call of duty",
		"ac valhalla": "assassin's creed valhalla",
		"elden ring":  "elden ring", // no synonym: unchanged
		"  bg3  ":     "baldur's gate 3",
	}
	for in, want := range cases {
		if got := expandQuery(in); got != want {
			t.Errorf("expandQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiscountPercent(t *testing.T) {
	msrp := 5999
	if got := discountPercent(3169, &msrp); got != 47 {
		t.Errorf("got %d, want 47", got)
	}
	if got := discountPercent(7000, &msrp); got != 0 {
		t.Errorf("price above retail should be 0%%, got %d", got)
	}
	if got := discountPercent(1000, nil); got != 0 {
		t.Errorf("no retail price should be 0%%, got %d", got)
	}
}
