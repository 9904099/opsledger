package discovery

import (
	"math/big"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/9904099/opsledger/internal/model"
)

func TestStringifyEC2TagsKeepsKeyValue(t *testing.T) {
	tags := stringifyEC2Tags([]*ec2.Tag{
		{Key: aws.String("Project"), Value: aws.String("business")},
		{Key: aws.String("Environment"), Value: aws.String("prod")},
		{Key: aws.String("EmptyValue"), Value: aws.String("")},
	})

	assertContains(t, tags, "tag:Project=business")
	assertContains(t, tags, "tag:Environment=prod")
	assertContains(t, tags, "tag:EmptyValue")
}

func TestBuildDiscoveredAssetPrefersResourceTags(t *testing.T) {
	asset := buildDiscoveredAsset(model.CloudAccount{
		Name:        "example-dev",
		Environment: "dev",
	}, discoveredAssetInput{
		Name:         "business-prod-api",
		ResourceType: "EC2",
		Tags:         []string{"aws", "ec2", "tag:Project=business", "tag:Environment=prod"},
	})

	if asset.ProjectCode != "business" {
		t.Fatalf("project_code = %q, want business", asset.ProjectCode)
	}
	if asset.Environment != "prod" {
		t.Fatalf("environment = %q, want prod", asset.Environment)
	}
}

func TestBuildDiscoveredAssetMixedAccountDoesNotInheritEnvironment(t *testing.T) {
	asset := buildDiscoveredAsset(model.CloudAccount{
		Name:        "shared-account",
		Environment: "mixed",
	}, discoveredAssetInput{
		Name:         "unlabeled-shared-host",
		ResourceType: "EC2",
		Tags:         []string{"aws", "ec2"},
	})

	if asset.ProjectCode != "public" {
		t.Fatalf("project_code = %q, want public", asset.ProjectCode)
	}
	if asset.Environment != "unknown" {
		t.Fatalf("environment = %q, want unknown", asset.Environment)
	}
}

func TestBuildDiscoveredAssetTagAliasesAndTextInference(t *testing.T) {
	asset := buildDiscoveredAsset(model.CloudAccount{
		Name:        "example-prod",
		Environment: "prod",
	}, discoveredAssetInput{
		Name:         "edge-local-gateway",
		ResourceType: "EC2",
		Tags:         []string{"tag:Project_Code=edge", "tag:Stage=local"},
	})

	if asset.ProjectCode != "edge" {
		t.Fatalf("project_code = %q, want edge", asset.ProjectCode)
	}
	if asset.Environment != "local" {
		t.Fatalf("environment = %q, want local", asset.Environment)
	}
}

func TestCostComparableLastMonthEndAndDelta(t *testing.T) {
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	lastMonthEnd := thisMonthStart
	end := comparableLastMonthEnd(now, lastMonthStart, lastMonthEnd)
	if got, want := end.Format("2006-01-02"), "2026-03-01"; got != want {
		t.Fatalf("comparable end = %s, want %s", got, want)
	}

	currentMonth := big.NewRat(120, 1)
	lastMonthToDate := big.NewRat(100, 1)
	delta := new(big.Rat).Sub(currentMonth, lastMonthToDate)
	if got, want := formatMoney(delta), "20.00"; got != want {
		t.Fatalf("delta money = %s, want %s", got, want)
	}

	forecast := forecastMonthCost(big.NewRat(100, 1), 10, 30)
	if got, want := formatMoney(forecast), "300.00"; got != want {
		t.Fatalf("forecast = %s, want %s", got, want)
	}
}

func assertContains(t *testing.T, values []string, wanted string) {
	t.Helper()
	for _, value := range values {
		if value == wanted {
			return
		}
	}
	t.Fatalf("%q not found in %#v", wanted, values)
}
