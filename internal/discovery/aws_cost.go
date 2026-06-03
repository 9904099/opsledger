package discovery

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

func (i *AWSImporter) SyncCloudAccountCost(ctx context.Context, cloudAccountID string) (model.CloudAccountCostResult, error) {
	account, err := i.store.GetCloudAccount(ctx, cloudAccountID)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	if strings.ToLower(account.PlatformCode) != "aws" {
		return model.CloudAccountCostResult{}, fmt.Errorf("cloud account %s is not an AWS account", account.Name)
	}

	rawAccount, ok := i.store.(*store.DBStore)
	if !ok {
		return model.CloudAccountCostResult{}, fmt.Errorf("unsupported store implementation")
	}

	accessKeyID, secretAccessKey, err := rawAccount.GetCloudAccountSecrets(ctx, account.ID)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	if strings.TrimSpace(accessKeyID) == "" || strings.TrimSpace(secretAccessKey) == "" {
		return model.CloudAccountCostResult{}, fmt.Errorf("cloud account %s has no usable AWS credentials", account.Name)
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(defaultAWSRegion),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}

	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := thisMonthStart.AddDate(0, 1, 0)
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	lastMonthEnd := thisMonthStart
	lastMonthToDateEnd := comparableLastMonthEnd(now, lastMonthStart, lastMonthEnd)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	currentEnd := today.AddDate(0, 0, 1)
	if currentEnd.After(nextMonthStart) {
		currentEnd = nextMonthStart
	}

	client := costexplorer.New(sess)
	startedAt := time.Now().Format(time.RFC3339)
	lastMonth, currency, err := getCostAndUsage(ctx, client, lastMonthStart, thisMonthStart)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	lastMonthToDate, comparableCurrency, err := getCostAndUsage(ctx, client, lastMonthStart, lastMonthToDateEnd)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	currentMonth, currentCurrency, err := getCostAndUsage(ctx, client, thisMonthStart, currentEnd)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	if currency == "" {
		currency = comparableCurrency
	}
	if currency == "" {
		currency = currentCurrency
	}

	forecast := forecastMonthCost(currentMonth, now.Day(), daysInMonth(now))
	delta := new(big.Rat).Sub(currentMonth, lastMonthToDate)
	result := model.CloudAccountCostResult{
		CloudAccountID:      account.ID,
		CloudAccountName:    account.Name,
		PlatformCode:        account.PlatformCode,
		Currency:            firstNonEmptyString(currency, "USD"),
		LastMonthCost:       formatMoney(lastMonth),
		LastMonthToDateCost: formatMoney(lastMonthToDate),
		CurrentMonthCost:    formatMoney(currentMonth),
		ForecastMonthCost:   formatMoney(forecast),
		MonthOverMonthDelta: formatMoney(delta),
		LastMonthStart:      lastMonthStart.Format("2006-01-02"),
		LastMonthEnd:        lastMonthEnd.Format("2006-01-02"),
		CurrentMonthStart:   thisMonthStart.Format("2006-01-02"),
		CurrentMonthEnd:     currentEnd.Format("2006-01-02"),
		StartedAt:           startedAt,
		FinishedAt:          time.Now().Format(time.RFC3339),
	}
	result.Summary = fmt.Sprintf("上月整月 %s %s，上月同进度 %s %s，本月 %s %s，预计 %s %s，同进度差额 %s %s",
		result.Currency, result.LastMonthCost, result.Currency, result.LastMonthToDateCost, result.Currency, result.CurrentMonthCost,
		result.Currency, result.ForecastMonthCost, result.Currency, result.MonthOverMonthDelta)

	if err := i.store.SetCloudAccountCostResult(ctx, account.ID, result); err != nil {
		return model.CloudAccountCostResult{}, err
	}
	records, err := collectCostRecords(ctx, client, account.ID, lastMonthStart, currentEnd, thisMonthStart, result.Currency, result.FinishedAt)
	if err != nil {
		return model.CloudAccountCostResult{}, err
	}
	if err := i.store.UpsertCloudAccountCostRecords(ctx, records); err != nil {
		return model.CloudAccountCostResult{}, err
	}
	return result, nil
}

func getCostAndUsage(ctx context.Context, client *costexplorer.CostExplorer, start, end time.Time) (*big.Rat, string, error) {
	output, err := client.GetCostAndUsageWithContext(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: aws.String(costexplorer.GranularityMonthly),
		Metrics:     []*string{aws.String("UnblendedCost")},
	})
	if err != nil {
		return nil, "", err
	}

	total := new(big.Rat)
	currency := ""
	for _, item := range output.ResultsByTime {
		if item == nil || item.Total == nil || item.Total["UnblendedCost"] == nil {
			continue
		}
		metric := item.Total["UnblendedCost"]
		if metric.Unit != nil && currency == "" {
			currency = aws.StringValue(metric.Unit)
		}
		amount, ok := new(big.Rat).SetString(aws.StringValue(metric.Amount))
		if ok {
			total.Add(total, amount)
		}
	}
	return total, currency, nil
}

func collectCostRecords(ctx context.Context, client *costexplorer.CostExplorer, accountID string, historicalStart, currentEnd, thisMonthStart time.Time, fallbackCurrency string, syncedAt string) ([]model.CloudAccountCostRecord, error) {
	var records []model.CloudAccountCostRecord
	appendRecords := func(items []model.CloudAccountCostRecord) {
		records = append(records, items...)
	}

	dailyTotal, err := getCostRecords(ctx, client, accountID, historicalStart, currentEnd, costexplorer.GranularityDaily, "", "total", fallbackCurrency, syncedAt)
	if err != nil {
		return nil, err
	}
	appendRecords(dailyTotal)

	dailyService, err := getCostRecords(ctx, client, accountID, thisMonthStart, currentEnd, costexplorer.GranularityDaily, "SERVICE", "service", fallbackCurrency, syncedAt)
	if err != nil {
		return nil, err
	}
	appendRecords(dailyService)

	monthlyService, err := getCostRecords(ctx, client, accountID, thisMonthStart, currentEnd, costexplorer.GranularityMonthly, "SERVICE", "service", fallbackCurrency, syncedAt)
	if err != nil {
		return nil, err
	}
	appendRecords(monthlyService)

	monthlyTotal, err := getCostRecords(ctx, client, accountID, thisMonthStart, currentEnd, costexplorer.GranularityMonthly, "", "total", fallbackCurrency, syncedAt)
	if err != nil {
		return nil, err
	}
	appendRecords(monthlyTotal)

	return records, nil
}

func getCostRecords(ctx context.Context, client *costexplorer.CostExplorer, accountID string, start, end time.Time, granularity string, groupByKey string, dimensionType string, fallbackCurrency string, syncedAt string) ([]model.CloudAccountCostRecord, error) {
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: aws.String(granularity),
		Metrics:     []*string{aws.String("UnblendedCost")},
	}
	if strings.TrimSpace(groupByKey) != "" {
		input.GroupBy = []*costexplorer.GroupDefinition{{
			Type: aws.String(costexplorer.GroupDefinitionTypeDimension),
			Key:  aws.String(groupByKey),
		}}
	}

	var records []model.CloudAccountCostRecord
	for {
		output, err := client.GetCostAndUsageWithContext(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, item := range output.ResultsByTime {
			if item == nil || item.TimePeriod == nil {
				continue
			}
			periodStart := aws.StringValue(item.TimePeriod.Start)
			periodEnd := aws.StringValue(item.TimePeriod.End)
			if strings.TrimSpace(groupByKey) == "" {
				metric := item.Total["UnblendedCost"]
				if metric == nil {
					continue
				}
				records = append(records, costRecord(accountID, periodStart, periodEnd, granularity, dimensionType, "total", metric, fallbackCurrency, syncedAt))
				continue
			}
			for _, group := range item.Groups {
				if group == nil || group.Metrics == nil || len(group.Keys) == 0 {
					continue
				}
				metric := group.Metrics["UnblendedCost"]
				if metric == nil {
					continue
				}
				name := aws.StringValue(group.Keys[0])
				if strings.TrimSpace(name) == "" {
					name = "Unknown"
				}
				records = append(records, costRecord(accountID, periodStart, periodEnd, granularity, dimensionType, name, metric, fallbackCurrency, syncedAt))
			}
		}
		if output.NextPageToken == nil || strings.TrimSpace(aws.StringValue(output.NextPageToken)) == "" {
			break
		}
		input.NextPageToken = output.NextPageToken
	}
	return records, nil
}

func costRecord(accountID string, periodStart string, periodEnd string, granularity string, dimensionType string, dimensionName string, metric *costexplorer.MetricValue, fallbackCurrency string, syncedAt string) model.CloudAccountCostRecord {
	currency := fallbackCurrency
	if metric.Unit != nil && strings.TrimSpace(aws.StringValue(metric.Unit)) != "" {
		currency = aws.StringValue(metric.Unit)
	}
	amount := "0.00"
	if parsed, ok := new(big.Rat).SetString(aws.StringValue(metric.Amount)); ok {
		amount = formatMoney(parsed)
	}
	return model.CloudAccountCostRecord{
		CloudAccountID: accountID,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Granularity:    strings.ToLower(granularity),
		DimensionType:  strings.ToLower(dimensionType),
		DimensionName:  dimensionName,
		Currency:       firstNonEmptyString(currency, "USD"),
		Amount:         amount,
		Source:         "aws_cost_explorer",
		SyncedAt:       syncedAt,
	}
}

func comparableLastMonthEnd(now time.Time, lastMonthStart, lastMonthEnd time.Time) time.Time {
	day := now.Day()
	lastMonthDays := daysInMonth(lastMonthStart)
	if day > lastMonthDays {
		day = lastMonthDays
	}
	end := time.Date(lastMonthStart.Year(), lastMonthStart.Month(), day, 0, 0, 0, 0, lastMonthStart.Location()).AddDate(0, 0, 1)
	if end.After(lastMonthEnd) {
		return lastMonthEnd
	}
	return end
}

func forecastMonthCost(current *big.Rat, elapsedDays, monthDays int) *big.Rat {
	if elapsedDays <= 0 {
		elapsedDays = 1
	}
	return new(big.Rat).Mul(current, big.NewRat(int64(monthDays), int64(elapsedDays)))
}

func daysInMonth(now time.Time) int {
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return first.AddDate(0, 1, -1).Day()
}

func formatMoney(value *big.Rat) string {
	if value == nil {
		return "0.00"
	}
	return value.FloatString(2)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
