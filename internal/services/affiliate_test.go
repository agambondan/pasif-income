package services

import (
	"strings"
	"testing"
)

func TestAffiliateBuildUsesCatalog(t *testing.T) {
	t.Setenv("AFFILIATE_CATALOG_JSON", `[
		{"title":"Stoic Journal","url":"https://shop.example/stoic-journal","cta":"Write one page today.","disclosure":"Disclosure: affiliate link.","pin_comment":"Journal link below.","tags":["stoicism","mindset"]},
		{"title":"Workout Bands","url":"https://shop.example/bands","cta":"Train daily.","disclosure":"Disclosure: affiliate link.","pin_comment":"Bands link below.","tags":["fitness"]}
	]`)

	svc := NewAffiliateService()
	plan := svc.Build("stoicism", "how to stay calm")
	if plan == nil {
		t.Fatal("expected affiliate plan")
	}
	if plan.Product.Title != "Stoic Journal" {
		t.Fatalf("expected stoic product, got %#v", plan.Product)
	}
	if !strings.Contains(plan.Description, "Disclosure") || !strings.Contains(plan.Description, plan.Product.URL) {
		t.Fatalf("expected composed description with disclosure and url, got %q", plan.Description)
	}
	if plan.PinComment == "" {
		t.Fatalf("expected pin comment, got %#v", plan)
	}
}

func TestAffiliateBuildFallback(t *testing.T) {
	t.Setenv("AFFILIATE_CATALOG_JSON", "")
	svc := NewAffiliateService()
	plan := svc.Build("finance", "budget discipline")
	if plan == nil {
		t.Fatal("expected fallback affiliate plan")
	}
	if !strings.Contains(strings.ToLower(plan.Product.Title), "budget") && !strings.Contains(strings.ToLower(plan.Product.Title), "stoic") {
		t.Fatalf("expected fallback product, got %#v", plan.Product)
	}
}
