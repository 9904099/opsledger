package discovery

import (
	"encoding/json"
	"testing"

	"github.com/9904099/opsledger/internal/model"
)

func TestParseCloudflareR2Buckets(t *testing.T) {
	wrapped, err := parseCloudflareR2Buckets(json.RawMessage(`{"buckets":[{"name":"assets","creation_date":"2026-06-01T00:00:00Z","location":"APAC"}]}`))
	if err != nil {
		t.Fatalf("wrapped parse failed: %v", err)
	}
	if len(wrapped) != 1 || wrapped[0].Name != "assets" || wrapped[0].Location != "APAC" {
		t.Fatalf("unexpected wrapped buckets: %#v", wrapped)
	}

	direct, err := parseCloudflareR2Buckets(json.RawMessage(`[{"name":"logs","storage_class":"standard"}]`))
	if err != nil {
		t.Fatalf("direct parse failed: %v", err)
	}
	if len(direct) != 1 || direct[0].Name != "logs" || direct[0].StorageClass != "standard" {
		t.Fatalf("unexpected direct buckets: %#v", direct)
	}
}

func TestCloudflareExtendedAssets(t *testing.T) {
	account := model.CloudAccount{
		ID:            "cloud-account-1",
		PlatformID:    "platform-cloudflare",
		PlatformCode:  "cloudflare",
		PlatformName:  "Cloudflare",
		Name:          "cloudflare-example",
		AccountID:     "example",
		Environment:   "prod",
		Owner:         "Ops",
		Criticality:   "medium",
		DefaultRegion: "global",
	}
	cfAccount := cloudflareAccount{ID: "cf-account-1", Name: "Example"}
	zone := cloudflareZone{ID: "zone-1", Name: "example.com", Status: "active"}

	worker := cloudflareWorkerAsset(account, cfAccount, cloudflareWorkerScript{
		ID:         "api-worker",
		UsageModel: "bundled",
		ModifiedOn: "2026-06-01T01:00:00Z",
	}, "2026-06-01")
	if worker.ResourceType != "Worker" || worker.Category != "edge" {
		t.Fatalf("unexpected worker asset: %#v", worker)
	}
	if worker.ExternalID != "cloudflare:worker:cf-account-1:api-worker" {
		t.Fatalf("worker external id = %q", worker.ExternalID)
	}
	if worker.Specs["usage_model"] != "bundled" {
		t.Fatalf("worker usage_model missing: %#v", worker.Specs)
	}

	r2 := cloudflareR2BucketAsset(account, cfAccount, cloudflareR2Bucket{
		Name:         "example-logs",
		Location:     "APAC",
		StorageClass: "standard",
	}, "2026-06-01")
	if r2.ResourceType != "R2 Bucket" || r2.Category != "storage" || r2.Region != "APAC" {
		t.Fatalf("unexpected r2 asset: %#v", r2)
	}
	if r2.Specs["storage_class"] != "standard" {
		t.Fatalf("r2 storage_class missing: %#v", r2.Specs)
	}

	waf := cloudflareWAFRulesetAsset(account, zone, cloudflareWAFRuleset{
		ID:          "ruleset-1",
		Name:        "default waf",
		Description: "managed rules",
		Kind:        "zone",
		Phase:       "http_request_firewall_managed",
		Version:     "12",
		Rules:       []cloudflareRulesetRule{{ID: "rule-1", Enabled: true}},
	}, "2026-06-01")
	if waf.ResourceType != "WAF Ruleset" || waf.Category != "security" || waf.Status != "active" {
		t.Fatalf("unexpected waf asset: %#v", waf)
	}
	if waf.Specs["enabled_rules"] != "1" || waf.Specs["rules_count"] != "1" {
		t.Fatalf("unexpected waf specs: %#v", waf.Specs)
	}

	lb := cloudflareLoadBalancerAsset(account, zone, cloudflareLoadBalancer{
		ID:              "lb-1",
		Name:            "api.example.com",
		Enabled:         true,
		Proxied:         true,
		SteeringPolicy:  "dynamic_latency",
		SessionAffinity: "cookie",
		DefaultPools:    []string{"pool-1", "pool-2"},
		FallbackPool:    "pool-1",
	}, "2026-06-01")
	if lb.ResourceType != "Load Balancer" || lb.Category != "network" || lb.Status != "active" {
		t.Fatalf("unexpected lb asset: %#v", lb)
	}
	if lb.Specs["steering_policy"] != "dynamic_latency" || lb.Specs["session_affinity"] != "cookie" {
		t.Fatalf("unexpected lb specs: %#v", lb.Specs)
	}
}
