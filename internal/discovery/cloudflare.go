package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

const cloudflareAPIBase = "https://api.cloudflare.com/client/v4"

type CloudflareImporter struct {
	store  store.Store
	client *http.Client
}

func NewCloudflareImporter(dataStore store.Store) *CloudflareImporter {
	return &CloudflareImporter{
		store: dataStore,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (i *CloudflareImporter) SyncCloudAccount(ctx context.Context, req model.CloudAccountSyncRequest) (model.CloudAccountSyncResult, error) {
	if strings.TrimSpace(req.CloudAccountID) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud_account_id is required")
	}

	account, err := i.store.GetCloudAccount(ctx, req.CloudAccountID)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if strings.ToLower(account.PlatformCode) != "cloudflare" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s is not a Cloudflare account", account.Name)
	}

	rawStore, ok := i.store.(*store.DBStore)
	if !ok {
		return model.CloudAccountSyncResult{}, fmt.Errorf("unsupported store implementation")
	}

	apiToken, _, err := rawStore.GetCloudAccountSecrets(ctx, account.ID)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if strings.TrimSpace(apiToken) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s has no usable Cloudflare API token", account.Name)
	}

	startedAt := time.Now().Format(time.RFC3339)
	result, err := i.importWithToken(ctx, account, apiToken)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	result.CloudAccountID = account.ID
	result.CloudAccountName = account.Name
	result.PlatformCode = account.PlatformCode
	result.StartedAt = startedAt
	result.FinishedAt = time.Now().Format(time.RFC3339)

	if err := i.store.SetCloudAccountSyncResult(ctx, account.ID, result); err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if err := i.store.RecordCloudAccountSync(ctx, model.CloudAccountSyncRecord{
		CloudAccountID:   account.ID,
		StartedAt:        result.StartedAt,
		FinishedAt:       result.FinishedAt,
		Status:           "success",
		DiscoveredAssets: result.DiscoveredAssets,
		CreatedAssets:    result.CreatedAssets,
		UpdatedAssets:    result.UpdatedAssets,
		Warnings:         result.Warnings,
		Breakdown:        result.ResourceBreakdown,
		Summary:          fmt.Sprintf("发现 %d 条，新增 %d 条，更新 %d 条", result.DiscoveredAssets, result.CreatedAssets, result.UpdatedAssets),
	}); err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	return result, nil
}

func (i *CloudflareImporter) importWithToken(ctx context.Context, account model.CloudAccount, apiToken string) (model.CloudAccountSyncResult, error) {
	today := time.Now().Format("2006-01-02")
	result := model.CloudAccountSyncResult{
		AccountID:         firstNonEmpty(account.AccountID, account.Name),
		Regions:           []string{"global"},
		ResourceBreakdown: map[string]int{},
	}

	zones, err := i.listZones(ctx, apiToken)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	var discovered []model.Asset
	var warnings []string
	accounts, err := i.listAccounts(ctx, apiToken)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("accounts: %s", err.Error()))
	} else {
		for _, cfAccount := range accounts {
			workers, err := i.listWorkerScripts(ctx, apiToken, cfAccount.ID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("workers@%s: %s", cfAccount.Name, err.Error()))
			}
			for _, worker := range workers {
				discovered = append(discovered, cloudflareWorkerAsset(account, cfAccount, worker, today))
			}

			buckets, err := i.listR2Buckets(ctx, apiToken, cfAccount.ID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("r2@%s: %s", cfAccount.Name, err.Error()))
			}
			for _, bucket := range buckets {
				discovered = append(discovered, cloudflareR2BucketAsset(account, cfAccount, bucket, today))
			}
		}
	}

	for _, zone := range zones {
		expiration, err := i.lookupDomainExpiration(ctx, zone.Name)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("rdap@%s: %s", zone.Name, err.Error()))
		}
		discovered = append(discovered, cloudflareZoneAsset(account, zone, today, expiration))

		records, err := i.listDNSRecords(ctx, apiToken, zone.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("dns-records@%s: %s", zone.Name, err.Error()))
			continue
		}
		for _, record := range records {
			discovered = append(discovered, cloudflareDNSRecordAsset(account, zone, record, today))
		}

		rulesets, err := i.listWAFRulesets(ctx, apiToken, zone.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("waf@%s: %s", zone.Name, err.Error()))
		}
		for _, ruleset := range rulesets {
			discovered = append(discovered, cloudflareWAFRulesetAsset(account, zone, ruleset, today))
		}

		loadBalancers, err := i.listLoadBalancers(ctx, apiToken, zone.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load-balancers@%s: %s", zone.Name, err.Error()))
		}
		for _, lb := range loadBalancers {
			discovered = append(discovered, cloudflareLoadBalancerAsset(account, zone, lb, today))
		}
	}

	result.DiscoveredAssets = len(discovered)
	result.Warnings = warnings
	for _, asset := range discovered {
		_, created, err := i.store.UpsertAssetBySource(ctx, asset)
		if err != nil {
			return model.CloudAccountSyncResult{}, err
		}
		result.ResourceBreakdown[asset.ResourceType]++
		if created {
			result.CreatedAssets++
		} else {
			result.UpdatedAssets++
		}
	}

	return result, nil
}

func (i *CloudflareImporter) listAccounts(ctx context.Context, token string) ([]cloudflareAccount, error) {
	var all []cloudflareAccount
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/accounts?per_page=50&page=%d", cloudflareAPIBase, page)
		var response cloudflareListResponse[cloudflareAccount]
		if err := i.do(ctx, token, endpoint, &response); err != nil {
			return nil, err
		}
		if err := responseError(response.Success, response.Errors); err != nil {
			return nil, err
		}
		all = append(all, response.Result...)
		if response.ResultInfo.Page >= response.ResultInfo.TotalPages || len(response.Result) == 0 {
			break
		}
	}
	return all, nil
}

func (i *CloudflareImporter) listZones(ctx context.Context, token string) ([]cloudflareZone, error) {
	var all []cloudflareZone
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/zones?per_page=50&page=%d", cloudflareAPIBase, page)
		var response cloudflareListResponse[cloudflareZone]
		if err := i.do(ctx, token, endpoint, &response); err != nil {
			return nil, err
		}
		if err := responseError(response.Success, response.Errors); err != nil {
			return nil, err
		}
		all = append(all, response.Result...)
		if response.ResultInfo.Page >= response.ResultInfo.TotalPages || len(response.Result) == 0 {
			break
		}
	}
	return all, nil
}

func (i *CloudflareImporter) listWorkerScripts(ctx context.Context, token, accountID string) ([]cloudflareWorkerScript, error) {
	endpoint := fmt.Sprintf("%s/accounts/%s/workers/scripts", cloudflareAPIBase, url.PathEscape(accountID))
	var response cloudflareListResponse[cloudflareWorkerScript]
	if err := i.do(ctx, token, endpoint, &response); err != nil {
		return nil, err
	}
	if err := responseError(response.Success, response.Errors); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (i *CloudflareImporter) listR2Buckets(ctx context.Context, token, accountID string) ([]cloudflareR2Bucket, error) {
	endpoint := fmt.Sprintf("%s/accounts/%s/r2/buckets", cloudflareAPIBase, url.PathEscape(accountID))
	var response cloudflareR2BucketResponse
	if err := i.do(ctx, token, endpoint, &response); err != nil {
		return nil, err
	}
	if err := responseError(response.Success, response.Errors); err != nil {
		return nil, err
	}
	return parseCloudflareR2Buckets(response.Result)
}

func (i *CloudflareImporter) listDNSRecords(ctx context.Context, token, zoneID string) ([]cloudflareDNSRecord, error) {
	var all []cloudflareDNSRecord
	escapedZoneID := url.PathEscape(zoneID)
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/zones/%s/dns_records?per_page=100&page=%d", cloudflareAPIBase, escapedZoneID, page)
		var response cloudflareListResponse[cloudflareDNSRecord]
		if err := i.do(ctx, token, endpoint, &response); err != nil {
			return nil, err
		}
		if err := responseError(response.Success, response.Errors); err != nil {
			return nil, err
		}
		all = append(all, response.Result...)
		if response.ResultInfo.Page >= response.ResultInfo.TotalPages || len(response.Result) == 0 {
			break
		}
	}
	return all, nil
}

func (i *CloudflareImporter) listWAFRulesets(ctx context.Context, token, zoneID string) ([]cloudflareWAFRuleset, error) {
	endpoint := fmt.Sprintf("%s/zones/%s/rulesets", cloudflareAPIBase, url.PathEscape(zoneID))
	var response cloudflareListResponse[cloudflareWAFRuleset]
	if err := i.do(ctx, token, endpoint, &response); err != nil {
		return nil, err
	}
	if err := responseError(response.Success, response.Errors); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (i *CloudflareImporter) listLoadBalancers(ctx context.Context, token, zoneID string) ([]cloudflareLoadBalancer, error) {
	var all []cloudflareLoadBalancer
	escapedZoneID := url.PathEscape(zoneID)
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/zones/%s/load_balancers?per_page=50&page=%d", cloudflareAPIBase, escapedZoneID, page)
		var response cloudflareListResponse[cloudflareLoadBalancer]
		if err := i.do(ctx, token, endpoint, &response); err != nil {
			return nil, err
		}
		if err := responseError(response.Success, response.Errors); err != nil {
			return nil, err
		}
		all = append(all, response.Result...)
		if response.ResultInfo.Page >= response.ResultInfo.TotalPages || len(response.Result) == 0 {
			break
		}
	}
	return all, nil
}

func (i *CloudflareImporter) do(ctx context.Context, token, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	request.Header.Set("Accept", "application/json")

	response, err := i.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var failure cloudflareErrorResponse
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if len(failure.Errors) > 0 {
			return fmt.Errorf("cloudflare api %d: %s", response.StatusCode, failure.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare api %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

func (i *CloudflareImporter) lookupDomainExpiration(ctx context.Context, domain string) (domainExpiration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://rdap.org/domain/"+url.PathEscape(strings.TrimSpace(domain)), nil)
	if err != nil {
		return domainExpiration{}, err
	}
	request.Header.Set("Accept", "application/rdap+json, application/json")
	request.Header.Set("User-Agent", "opsledger-rdap/1.0")

	response, err := i.client.Do(request)
	if err != nil {
		return domainExpiration{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return domainExpiration{}, fmt.Errorf("rdap http %d", response.StatusCode)
	}

	var data rdapDomainResponse
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return domainExpiration{}, err
	}
	for _, event := range data.Events {
		action := strings.ToLower(strings.TrimSpace(event.EventAction))
		if action != "expiration" && action != "registration expiration" && action != "expires" {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, event.EventDate)
		if err != nil {
			continue
		}
		return domainExpiration{
			ExpiresAt:     expiresAt.Format("2006-01-02"),
			ExpiresInDays: int(time.Until(expiresAt).Hours() / 24),
		}, nil
	}
	return domainExpiration{}, fmt.Errorf("expiration event not found")
}

func cloudflareZoneAsset(account model.CloudAccount, zone cloudflareZone, lastCheckedAt string, expiration domainExpiration) model.Asset {
	specs := map[string]string{
		"zone_id":      zone.ID,
		"name":         zone.Name,
		"status":       zone.Status,
		"plan":         zone.Plan.Name,
		"type":         zone.Type,
		"paused":       fmt.Sprintf("%t", zone.Paused),
		"name_servers": strings.Join(zone.NameServers, ", "),
	}
	if expiration.ExpiresAt != "" {
		specs["expires_at"] = expiration.ExpiresAt
		specs["expires_in_days"] = strconv.Itoa(expiration.ExpiresInDays)
	}
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(account.AccountID, account.Name),
		Category:         "edge",
		ResourceType:     "Zone",
		Region:           "global",
		Environment:      account.Environment,
		Name:             zone.Name,
		Endpoint:         zone.Name,
		Owner:            account.Owner,
		Status:           cloudflareStatus(zone.Status),
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             []string{"cloudflare", "zone"},
		Notes:            fmt.Sprintf("plan=%s; paused=%t; type=%s", zone.Plan.Name, zone.Paused, zone.Type),
		Specs:            specs,
		Source:           "cloudflare",
		ExternalID:       "cloudflare:zone:" + zone.ID,
	}
}

func cloudflareDNSRecordAsset(account model.CloudAccount, zone cloudflareZone, record cloudflareDNSRecord, lastCheckedAt string) model.Asset {
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(account.AccountID, account.Name),
		Category:         "dns",
		ResourceType:     "DNS Record",
		Region:           "global",
		Environment:      account.Environment,
		Name:             record.Name,
		Endpoint:         record.Content,
		Owner:            account.Owner,
		Status:           "active",
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             []string{"cloudflare", "dns", strings.ToLower(record.Type)},
		Notes:            fmt.Sprintf("zone=%s; type=%s; proxied=%t; ttl=%d", zone.Name, record.Type, record.Proxied, record.TTL),
		Specs: map[string]string{
			"record_id": record.ID,
			"zone_id":   zone.ID,
			"zone":      zone.Name,
			"type":      record.Type,
			"name":      record.Name,
			"content":   record.Content,
			"proxied":   fmt.Sprintf("%t", record.Proxied),
			"ttl":       fmt.Sprintf("%d", record.TTL),
		},
		Source:     "cloudflare",
		ExternalID: "cloudflare:dns-record:" + record.ID,
	}
}

func cloudflareWorkerAsset(account model.CloudAccount, cfAccount cloudflareAccount, worker cloudflareWorkerScript, lastCheckedAt string) model.Asset {
	name := firstNonEmpty(worker.ID, worker.Name)
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(cfAccount.ID, account.AccountID, account.Name),
		Category:         "edge",
		ResourceType:     "Worker",
		Region:           "global",
		Environment:      account.Environment,
		Name:             name,
		Endpoint:         fmt.Sprintf("workers/%s", name),
		Owner:            account.Owner,
		Status:           "active",
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             []string{"cloudflare", "worker", "script"},
		Notes:            fmt.Sprintf("account=%s; created=%s; modified=%s", cfAccount.Name, worker.CreatedOn, worker.ModifiedOn),
		Specs: map[string]string{
			"account_id":  cfAccount.ID,
			"account":     cfAccount.Name,
			"script_id":   name,
			"etag":        worker.ETag,
			"usage_model": worker.UsageModel,
			"created_on":  worker.CreatedOn,
			"modified_on": worker.ModifiedOn,
		},
		Source:     "cloudflare",
		ExternalID: "cloudflare:worker:" + cfAccount.ID + ":" + name,
	}
}

func cloudflareR2BucketAsset(account model.CloudAccount, cfAccount cloudflareAccount, bucket cloudflareR2Bucket, lastCheckedAt string) model.Asset {
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(cfAccount.ID, account.AccountID, account.Name),
		Category:         "storage",
		ResourceType:     "R2 Bucket",
		Region:           firstNonEmpty(bucket.Location, "global"),
		Environment:      account.Environment,
		Name:             bucket.Name,
		Endpoint:         fmt.Sprintf("r2://%s", bucket.Name),
		Owner:            account.Owner,
		Status:           "active",
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             []string{"cloudflare", "r2", "bucket"},
		Notes:            fmt.Sprintf("account=%s; created=%s", cfAccount.Name, bucket.CreationDate),
		Specs: map[string]string{
			"account_id":    cfAccount.ID,
			"account":       cfAccount.Name,
			"name":          bucket.Name,
			"creation_date": bucket.CreationDate,
			"location":      bucket.Location,
			"storage_class": bucket.StorageClass,
		},
		Source:     "cloudflare",
		ExternalID: "cloudflare:r2:" + cfAccount.ID + ":" + bucket.Name,
	}
}

func cloudflareWAFRulesetAsset(account model.CloudAccount, zone cloudflareZone, ruleset cloudflareWAFRuleset, lastCheckedAt string) model.Asset {
	enabledRules := 0
	for _, rule := range ruleset.Rules {
		if rule.Enabled {
			enabledRules++
		}
	}
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(account.AccountID, account.Name),
		Category:         "security",
		ResourceType:     "WAF Ruleset",
		Region:           "global",
		Environment:      account.Environment,
		Name:             firstNonEmpty(ruleset.Name, ruleset.ID),
		Endpoint:         zone.Name,
		Owner:            account.Owner,
		Status:           cloudflareRulesetStatus(ruleset),
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             compactTags("cloudflare", "waf", ruleset.Phase, ruleset.Kind),
		Notes:            fmt.Sprintf("zone=%s; phase=%s; kind=%s; rules=%d", zone.Name, ruleset.Phase, ruleset.Kind, len(ruleset.Rules)),
		Specs: map[string]string{
			"ruleset_id":    ruleset.ID,
			"zone_id":       zone.ID,
			"zone":          zone.Name,
			"name":          ruleset.Name,
			"description":   ruleset.Description,
			"phase":         ruleset.Phase,
			"kind":          ruleset.Kind,
			"version":       ruleset.Version,
			"last_updated":  ruleset.LastUpdated,
			"rules_count":   strconv.Itoa(len(ruleset.Rules)),
			"enabled_rules": strconv.Itoa(enabledRules),
		},
		Source:     "cloudflare",
		ExternalID: "cloudflare:waf:" + zone.ID + ":" + ruleset.ID,
	}
}

func cloudflareLoadBalancerAsset(account model.CloudAccount, zone cloudflareZone, lb cloudflareLoadBalancer, lastCheckedAt string) model.Asset {
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(account.AccountID, account.Name),
		Category:         "network",
		ResourceType:     "Load Balancer",
		Region:           "global",
		Environment:      account.Environment,
		Name:             firstNonEmpty(lb.Name, lb.ID),
		Endpoint:         firstNonEmpty(lb.Name, zone.Name),
		Owner:            account.Owner,
		Status:           cloudflareEnabledStatus(lb.Enabled),
		Criticality:      account.Criticality,
		LastCheckedAt:    lastCheckedAt,
		Tags:             []string{"cloudflare", "load-balancer"},
		Notes:            fmt.Sprintf("zone=%s; enabled=%t; default_pools=%d; fallback_pool=%s", zone.Name, lb.Enabled, len(lb.DefaultPools), lb.FallbackPool),
		Specs: map[string]string{
			"load_balancer_id": lb.ID,
			"zone_id":          zone.ID,
			"zone":             zone.Name,
			"name":             lb.Name,
			"enabled":          fmt.Sprintf("%t", lb.Enabled),
			"proxied":          fmt.Sprintf("%t", lb.Proxied),
			"steering_policy":  lb.SteeringPolicy,
			"session_affinity": lb.SessionAffinity,
			"fallback_pool":    lb.FallbackPool,
			"default_pools":    strings.Join(lb.DefaultPools, ","),
			"pop_pools":        strings.Join(lb.PopPools, ","),
			"description":      lb.Description,
		},
		Source:     "cloudflare",
		ExternalID: "cloudflare:load-balancer:" + zone.ID + ":" + lb.ID,
	}
}

func cloudflareStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return "active"
	default:
		return "maintenance"
	}
}

func cloudflareEnabledStatus(enabled bool) string {
	if enabled {
		return "active"
	}
	return "maintenance"
}

func cloudflareRulesetStatus(ruleset cloudflareWAFRuleset) string {
	for _, rule := range ruleset.Rules {
		if rule.Enabled {
			return "active"
		}
	}
	if len(ruleset.Rules) == 0 && strings.TrimSpace(ruleset.Phase) != "" {
		return "active"
	}
	return "maintenance"
}

func compactTags(values ...string) []string {
	tags := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		tags = append(tags, value)
	}
	return tags
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func responseError(success bool, messages []cloudflareAPIMessage) error {
	if success {
		return nil
	}
	if len(messages) > 0 {
		return fmt.Errorf("cloudflare api: %s", messages[0].Message)
	}
	return fmt.Errorf("cloudflare api: request failed")
}

func parseCloudflareR2Buckets(raw json.RawMessage) ([]cloudflareR2Bucket, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var wrapped struct {
		Buckets []cloudflareR2Bucket `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Buckets != nil {
		return wrapped.Buckets, nil
	}

	var direct []cloudflareR2Bucket
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, err
	}
	return direct, nil
}

type cloudflareListResponse[T any] struct {
	Success    bool                   `json:"success"`
	Errors     []cloudflareAPIMessage `json:"errors"`
	Result     []T                    `json:"result"`
	ResultInfo cloudflareResultInfo   `json:"result_info"`
}

type cloudflareErrorResponse struct {
	Success bool                   `json:"success"`
	Errors  []cloudflareAPIMessage `json:"errors"`
}

type cloudflareR2BucketResponse struct {
	Success bool                   `json:"success"`
	Errors  []cloudflareAPIMessage `json:"errors"`
	Result  json.RawMessage        `json:"result"`
}

type cloudflareAPIMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cloudflareResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

type cloudflareAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareZone struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Status      string             `json:"status"`
	Paused      bool               `json:"paused"`
	Type        string             `json:"type"`
	NameServers []string           `json:"name_servers"`
	Plan        cloudflareZonePlan `json:"plan"`
}

type cloudflareZonePlan struct {
	Name string `json:"name"`
}

type cloudflareDNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type cloudflareWorkerScript struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ETag       string `json:"etag"`
	UsageModel string `json:"usage_model"`
	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on"`
}

type cloudflareR2Bucket struct {
	Name         string `json:"name"`
	CreationDate string `json:"creation_date"`
	Location     string `json:"location"`
	StorageClass string `json:"storage_class"`
}

type cloudflareWAFRuleset struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Kind        string                  `json:"kind"`
	Phase       string                  `json:"phase"`
	Version     string                  `json:"version"`
	LastUpdated string                  `json:"last_updated"`
	Rules       []cloudflareRulesetRule `json:"rules"`
}

type cloudflareRulesetRule struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Expression  string `json:"expression"`
}

type cloudflareLoadBalancer struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Enabled         bool     `json:"enabled"`
	Proxied         bool     `json:"proxied"`
	SteeringPolicy  string   `json:"steering_policy"`
	SessionAffinity string   `json:"session_affinity"`
	FallbackPool    string   `json:"fallback_pool"`
	DefaultPools    []string `json:"default_pools"`
	PopPools        []string `json:"pop_pools"`
}

type domainExpiration struct {
	ExpiresAt     string
	ExpiresInDays int
}

type rdapDomainResponse struct {
	Events []rdapEvent `json:"events"`
}

type rdapEvent struct {
	EventAction string `json:"eventAction"`
	EventDate   string `json:"eventDate"`
}
