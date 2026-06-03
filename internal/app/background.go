package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) startAutoProbe() {
	interval := durationEnv("OPSLEDGER_PROBE_INTERVAL", 60*time.Second)
	minAge := durationEnv("OPSLEDGER_PROBE_MIN_AGE", 5*time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	s.probeCancel = cancel
	s.probeDone = make(chan struct{})

	go func() {
		defer close(s.probeDone)
		s.runAutoProbeCycle(ctx, minAge)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runAutoProbeCycle(ctx, minAge)
			}
		}
	}()
	log.Printf("auto probe enabled: interval=%s min_age=%s", interval, minAge)
}

func (s *Server) startAutoSync() {
	interval := durationEnv("OPSLEDGER_SYNC_CHECK_INTERVAL", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	s.syncCancel = cancel
	s.syncDone = make(chan struct{})

	go func() {
		defer close(s.syncDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runAutoSyncCycle(ctx)
			}
		}
	}()
	log.Printf("auto sync scheduler enabled: check_interval=%s", interval)
}

func (s *Server) runAutoSyncCycle(ctx context.Context) {
	accounts, err := s.store.ListCloudAccounts(ctx)
	if err != nil {
		log.Printf("auto sync list cloud accounts: %v", err)
		return
	}
	for _, account := range accounts {
		if !shouldAutoSyncAccount(account, time.Now()) {
			continue
		}
		account := account
		go s.runAutoSyncAccount(ctx, account)
	}
}

func (s *Server) runAutoSyncAccount(parent context.Context, account model.CloudAccount) {
	if !s.tryStartSync(account.ID) {
		return
	}
	defer s.finishSync(account.ID)

	timeout := durationEnv("OPSLEDGER_SYNC_TIMEOUT", 20*time.Minute)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	result, err := s.syncCloudAccount(ctx, account, model.CloudAccountSyncRequest{
		CloudAccountID: account.ID,
		Region:         account.DefaultRegion,
	})
	if err != nil {
		log.Printf("auto sync %s/%s failed: %v", account.PlatformCode, account.Name, err)
		s.recordSystemAudit("cloud_account.auto_sync", "cloud_account", account.ID, account.Name, "failed", err.Error(), map[string]string{"platform": account.PlatformCode})
		return
	}
	log.Printf("auto sync %s/%s: discovered=%d created=%d updated=%d stale=%d", account.PlatformCode, account.Name, result.DiscoveredAssets, result.CreatedAssets, result.UpdatedAssets, result.StaleAssets)
	s.recordSystemAudit("cloud_account.auto_sync", "cloud_account", account.ID, account.Name, "success", formatCloudAccountSyncSummary(result), map[string]string{
		"platform":          account.PlatformCode,
		"discovered_assets": fmt.Sprintf("%d", result.DiscoveredAssets),
		"created_assets":    fmt.Sprintf("%d", result.CreatedAssets),
		"updated_assets":    fmt.Sprintf("%d", result.UpdatedAssets),
		"stale_assets":      fmt.Sprintf("%d", result.StaleAssets),
	})
	if strings.EqualFold(account.PlatformCode, "aws") {
		costResult, err := s.awsImporter.SyncCloudAccountCost(ctx, account.ID)
		if err != nil {
			log.Printf("auto cost sync %s/%s failed: %v", account.PlatformCode, account.Name, err)
			s.recordSystemAudit("cloud_account.auto_cost_sync", "cloud_account", account.ID, account.Name, "failed", err.Error(), map[string]string{"platform": account.PlatformCode})
			return
		}
		log.Printf("auto cost sync %s/%s: current_month=%s %s forecast=%s", account.PlatformCode, account.Name, costResult.Currency, costResult.CurrentMonthCost, costResult.ForecastMonthCost)
		s.recordSystemAudit("cloud_account.auto_cost_sync", "cloud_account", account.ID, account.Name, "success", costResult.Summary, map[string]string{
			"platform":               account.PlatformCode,
			"current_month_cost":     costResult.CurrentMonthCost,
			"forecast_month_cost":    costResult.ForecastMonthCost,
			"month_over_month_delta": costResult.MonthOverMonthDelta,
		})
	}
}

func (s *Server) tryStartSync(accountID string) bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.syncRunning[accountID] {
		return false
	}
	s.syncRunning[accountID] = true
	return true
}

func (s *Server) finishSync(accountID string) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	delete(s.syncRunning, accountID)
}

func shouldAutoSyncAccount(account model.CloudAccount, now time.Time) bool {
	if !account.SyncEnabled || strings.EqualFold(account.SyncMode, "manual") {
		return false
	}
	interval := syncInterval(account.SyncMode, account.SyncCron)
	if interval <= 0 {
		return false
	}
	if strings.TrimSpace(account.LastSyncAt) == "" {
		return true
	}
	lastSyncAt, err := time.Parse(time.RFC3339, account.LastSyncAt)
	if err != nil {
		return true
	}
	return now.Sub(lastSyncAt) >= interval
}

func syncInterval(syncMode string, syncCron string) time.Duration {
	syncMode = strings.ToLower(strings.TrimSpace(syncMode))
	syncCron = strings.TrimSpace(syncCron)
	if syncMode == "manual" {
		return 0
	}
	if duration, err := time.ParseDuration(syncCron); err == nil && duration > 0 {
		return duration
	}
	if duration := composedInterval(syncCron); duration > 0 {
		return duration
	}
	if dailyTimeInterval(syncCron) > 0 {
		return 24 * time.Hour
	}
	if interval := cronLikeInterval(syncCron); interval > 0 {
		return interval
	}
	switch syncMode {
	case "interval", "auto", "scheduled", "cron":
		return 6 * time.Hour
	default:
		return 0
	}
}

func cronLikeInterval(expr string) time.Duration {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return 0
	}
	minute := fields[0]
	hour := fields[1]
	switch {
	case strings.HasPrefix(hour, "*/"):
		if value := positiveInt(strings.TrimPrefix(hour, "*/")); value > 0 {
			return time.Duration(value) * time.Hour
		}
	case strings.HasPrefix(minute, "*/"):
		if value := positiveInt(strings.TrimPrefix(minute, "*/")); value > 0 {
			return time.Duration(value) * time.Minute
		}
	case hour == "*" && minute != "*":
		return time.Hour
	case minute == "*" && hour == "*":
		return time.Minute
	}
	return 24 * time.Hour
}

func composedInterval(expr string) time.Duration {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0
	}
	units := []struct {
		suffix string
		value  time.Duration
	}{
		{suffix: "mo", value: 30 * 24 * time.Hour},
		{suffix: "y", value: 365 * 24 * time.Hour},
		{suffix: "w", value: 7 * 24 * time.Hour},
		{suffix: "d", value: 24 * time.Hour},
		{suffix: "h", value: time.Hour},
		{suffix: "m", value: time.Minute},
		{suffix: "s", value: time.Second},
	}
	remaining := expr
	var total time.Duration
	for remaining != "" {
		matched := false
		for _, unit := range units {
			index := strings.Index(remaining, unit.suffix)
			if index <= 0 {
				continue
			}
			value := positiveInt(remaining[:index])
			if value <= 0 {
				return 0
			}
			total += time.Duration(value) * unit.value
			remaining = remaining[index+len(unit.suffix):]
			matched = true
			break
		}
		if !matched {
			return 0
		}
	}
	return total
}

func dailyTimeInterval(expr string) time.Duration {
	parts := strings.Split(expr, ":")
	if len(parts) != 2 {
		return 0
	}
	hour, ok := clockPart(parts[0])
	if !ok || hour > 23 {
		return 0
	}
	minute, ok := clockPart(parts[1])
	if !ok || minute > 59 {
		return 0
	}
	return 24 * time.Hour
}

func clockPart(value string) (int, bool) {
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil || result < 0 {
		return 0, false
	}
	return result, true
}

func positiveInt(value string) int {
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil || result <= 0 {
		return 0
	}
	return result
}

func (s *Server) runAutoProbeCycle(ctx context.Context, minAge time.Duration) {
	assets, err := s.store.ListProbeAssets(ctx)
	if err != nil {
		log.Printf("auto probe list assets: %v", err)
		return
	}
	if len(assets) == 0 {
		return
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, asset := range assets {
		if !s.shouldProbeAsset(ctx, asset.ID, minAge) {
			continue
		}
		asset := asset
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			s.runAutoProbeAsset(ctx, asset)
		}()
	}
	wg.Wait()
}

func (s *Server) shouldProbeAsset(ctx context.Context, assetID string, minAge time.Duration) bool {
	latest, err := s.store.LatestProbe(ctx, assetID)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if err != nil {
		log.Printf("auto probe latest %s: %v", assetID, err)
		return true
	}
	checkedAt, err := time.Parse(time.RFC3339, latest.CheckedAt)
	if err != nil {
		return true
	}
	return time.Since(checkedAt) >= minAge
}

func (s *Server) runAutoProbeAsset(ctx context.Context, asset model.Asset) {
	probe, err := probeHTTP(ctx, asset)
	if err != nil {
		log.Printf("auto probe %s: %v", asset.Name, err)
		return
	}
	record, err := s.store.CreateProbe(ctx, probe)
	if err != nil {
		log.Printf("auto probe save %s: %v", asset.Name, err)
		return
	}
	if record.Status != "up" {
		if _, err := s.store.CreateInspection(ctx, model.InspectionRecord{
			AssetID:   asset.ID,
			Executor:  "auto-probe",
			Result:    "failed",
			Summary:   fmt.Sprintf("自动拨测异常：%s HTTP %d %dms %s", record.URL, record.StatusCode, record.LatencyMS, record.Error),
			CheckedAt: record.CheckedAt,
		}); err != nil {
			log.Printf("auto probe inspection %s: %v", asset.Name, err)
		}
	}
	s.reconcileProbeAlert(ctx, asset, record, "auto-probe")
	log.Printf("auto probe %s -> %s http=%d latency=%dms", asset.Name, record.Status, record.StatusCode, record.LatencyMS)
}

func (s *Server) reconcileProbeAlert(ctx context.Context, asset model.Asset, record model.ProbeRecord, executor string) {
	if record.Status == "up" {
		if err := s.store.ResolveOpenAlertsForAssetSource(ctx, asset.ID, "probe", executor, "拨测恢复，系统自动关闭"); err != nil {
			log.Printf("resolve probe alert %s: %v", asset.Name, err)
		}
		return
	}

	severity := "warning"
	if record.Status == "down" || record.StatusCode >= 500 || record.Error != "" {
		severity = "critical"
	}
	summary := fmt.Sprintf("%s HTTP %d %dms %s", record.URL, record.StatusCode, record.LatencyMS, strings.TrimSpace(record.Error))
	if _, err := s.store.UpsertAlert(ctx, model.AlertUpsertRequest{
		AssetID:  asset.ID,
		Source:   "probe",
		Severity: severity,
		Title:    "拨测异常",
		Summary:  summary,
		SeenAt:   record.CheckedAt,
	}); err != nil {
		log.Printf("upsert probe alert %s: %v", asset.Name, err)
	}
}

func probeHTTP(ctx context.Context, asset model.Asset) (model.ProbeRecord, error) {
	url := probeURL(asset)
	if url == "" {
		return model.ProbeRecord{}, errors.New("asset has no probeable URL")
	}

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return model.ProbeRecord{}, err
	}
	request.Header.Set("User-Agent", "opsledger-probe/1.0")

	started := time.Now()
	record := model.ProbeRecord{
		AssetID:   asset.ID,
		URL:       url,
		Method:    http.MethodGet,
		Status:    "failed",
		CheckedAt: started.Format(time.RFC3339),
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	response, err := client.Do(request)
	record.LatencyMS = int(time.Since(started).Milliseconds())
	if err != nil {
		record.Error = err.Error()
		return record, nil
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 512*1024))

	record.StatusCode = response.StatusCode
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		record.Status = "up"
	} else {
		record.Status = "down"
		record.Error = response.Status
	}

	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		expiresAt := response.TLS.PeerCertificates[0].NotAfter
		record.TLSExpiresAt = expiresAt.Format(time.RFC3339)
		record.CertDaysRemaining = int(time.Until(expiresAt).Hours() / 24)
	}
	return record, nil
}

func probeURL(asset model.Asset) string {
	candidates := []string{}
	if asset.ResourceType == "DNS Record" {
		if value := normalizeProbeHost(asset.Name); value != "" {
			candidates = append(candidates, value)
		}
	}
	candidates = append(candidates, normalizeProbeHost(asset.Endpoint), normalizeProbeHost(asset.Name))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.Contains(candidate, " ") || strings.HasPrefix(candidate, "\"") {
			continue
		}
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			return candidate
		}
		if strings.Contains(candidate, ".") && !strings.Contains(candidate, "/") {
			return "https://" + candidate
		}
	}
	return ""
}

func normalizeProbeHost(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "*.")
	value = strings.TrimSuffix(value, ".")
	if value == "" || strings.ContainsAny(value, "\" ") || strings.Contains(value, "_") {
		return ""
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q, fallback to %s", key, value, fallback)
		return fallback
	}
	return duration
}
