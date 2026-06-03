package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

const defaultAWSRegion = "us-east-1"

type AWSImporter struct {
	store store.Store
}

func NewAWSImporter(dataStore store.Store) *AWSImporter {
	return &AWSImporter{store: dataStore}
}

func (i *AWSImporter) SyncCloudAccount(ctx context.Context, req model.CloudAccountSyncRequest) (model.CloudAccountSyncResult, error) {
	if strings.TrimSpace(req.CloudAccountID) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud_account_id is required")
	}

	account, err := i.store.GetCloudAccount(ctx, req.CloudAccountID)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if strings.ToLower(account.PlatformCode) != "aws" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s is not an AWS account", account.Name)
	}

	rawAccount, ok := i.store.(*store.DBStore)
	if !ok {
		return model.CloudAccountSyncResult{}, fmt.Errorf("unsupported store implementation")
	}

	accessKeyID, secretAccessKey, err := rawAccount.GetCloudAccountSecrets(ctx, account.ID)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if strings.TrimSpace(accessKeyID) == "" || strings.TrimSpace(secretAccessKey) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s has no usable AWS credentials", account.Name)
	}

	region := strings.TrimSpace(req.Region)
	if region == "" {
		region = strings.TrimSpace(account.DefaultRegion)
	}
	if region == "" {
		region = defaultAWSRegion
	}

	startedAt := time.Now().Format(time.RFC3339)
	result, err := i.importWithCredentials(ctx, account, accessKeyID, secretAccessKey, region)
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
		Summary:          fmt.Sprintf("发现 %d 条，新增 %d 条，更新 %d 条，标记 stale %d 条", result.DiscoveredAssets, result.CreatedAssets, result.UpdatedAssets, result.StaleAssets),
	}); err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	return result, nil
}

func (i *AWSImporter) importWithCredentials(ctx context.Context, account model.CloudAccount, accessKeyID, secretAccessKey, bootstrapRegion string) (model.CloudAccountSyncResult, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(bootstrapRegion),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	accountID, err := getAWSAccountID(sess)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	regions, err := resolveRegions(sess, bootstrapRegion, account.DefaultRegion)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	result := model.CloudAccountSyncResult{
		AccountID:         accountID,
		Regions:           regions,
		ResourceBreakdown: map[string]int{},
	}

	today := time.Now().Format("2006-01-02")
	var discovered []model.Asset
	var warnings []string

	s3Assets, s3Warnings := discoverS3Assets(sess, account, accountID, bootstrapRegion, today)
	discovered = append(discovered, s3Assets...)
	warnings = append(warnings, s3Warnings...)

	for _, region := range regions {
		regionAssets, regionWarnings := discoverRegionalAssets(sess, account, accountID, region, today)
		discovered = append(discovered, regionAssets...)
		warnings = append(warnings, regionWarnings...)
	}

	result.DiscoveredAssets = len(discovered)
	result.Warnings = warnings

	activeExternalIDs := make([]string, 0, len(discovered))
	for _, asset := range discovered {
		if strings.TrimSpace(asset.ExternalID) != "" {
			activeExternalIDs = append(activeExternalIDs, asset.ExternalID)
		}
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
	staleAssets, err := i.store.MarkAssetsStaleBySource(ctx, account.ID, "aws", activeExternalIDs, result.Regions, today)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	result.StaleAssets = staleAssets

	return result, nil
}

func getAWSAccountID(sess *session.Session) (string, error) {
	client := sts.New(sess)
	identity, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return aws.StringValue(identity.Account), nil
}

func resolveRegions(sess *session.Session, requestedRegion, defaultRegion string) ([]string, error) {
	if strings.TrimSpace(requestedRegion) != "" && strings.TrimSpace(requestedRegion) != defaultAWSRegion {
		return []string{strings.TrimSpace(requestedRegion)}, nil
	}
	if strings.TrimSpace(defaultRegion) != "" && strings.TrimSpace(requestedRegion) != "" {
		return []string{strings.TrimSpace(defaultRegion)}, nil
	}

	client := ec2.New(sess)
	output, err := client.DescribeRegions(&ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	var regions []string
	for _, item := range output.Regions {
		name := aws.StringValue(item.RegionName)
		status := aws.StringValue(item.OptInStatus)
		if name == "" {
			continue
		}
		if status == "" || status == "opt-in-not-required" || status == "opted-in" {
			regions = append(regions, name)
		}
	}
	sort.Strings(regions)
	return regions, nil
}

func discoverRegionalAssets(baseSession *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, []string) {
	sess := baseSession.Copy(&aws.Config{Region: aws.String(region)})
	var assets []model.Asset
	var warnings []string

	appendWarn := func(service string, err error) {
		if err == nil {
			return
		}
		warnings = append(warnings, fmt.Sprintf("%s@%s: %s", service, region, err.Error()))
	}

	if items, err := discoverEC2Instances(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("ec2-instances", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverEBSVolumes(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("ebs", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverElasticIPs(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("eip", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverVPCs(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("vpc", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverSubnets(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("subnet", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverSecurityGroups(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("security-group", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverRDSInstances(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("rds", err)
	} else {
		assets = append(assets, items...)
	}

	if items, err := discoverLoadBalancers(sess, account, accountID, region, lastCheckedAt); err != nil {
		appendWarn("elbv2", err)
	} else {
		assets = append(assets, items...)
	}

	return assets, warnings
}

func discoverS3Assets(sess *session.Session, account model.CloudAccount, accountID, filterRegion, lastCheckedAt string) ([]model.Asset, []string) {
	client := s3.New(sess)
	output, err := client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, []string{fmt.Sprintf("s3-global: %s", err.Error())}
	}

	var assets []model.Asset
	var warnings []string
	for _, bucket := range output.Buckets {
		name := aws.StringValue(bucket.Name)
		if name == "" {
			continue
		}

		location, err := client.GetBucketLocation(&s3.GetBucketLocationInput{
			Bucket: aws.String(name),
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("s3-bucket-location:%s: %s", name, err.Error()))
			continue
		}

		region := normalizeS3Region(location.LocationConstraint)
		if strings.TrimSpace(filterRegion) != "" && strings.TrimSpace(filterRegion) != defaultAWSRegion && region != strings.TrimSpace(filterRegion) {
			continue
		}

		tagValues, err := getS3BucketTags(client, name)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("s3-bucket-tags:%s: %s", name, err.Error()))
		}

		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "storage",
			ResourceType:  "S3 Bucket",
			Name:          name,
			Endpoint:      fmt.Sprintf("s3://%s", name),
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:s3:" + name,
			Status:        "active",
			Tags:          mergeTags([]string{"aws", "s3", "bucket"}, tagValues),
			Notes:         "Auto discovered from AWS S3.",
			Specs: map[string]string{
				"bucket": name,
				"region": region,
			},
		}))
	}

	return assets, warnings
}

func discoverEC2Instances(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := aws.StringValue(instance.InstanceId)
			if instanceID == "" {
				continue
			}
			name := instanceID
			if tagName := findEC2Tag(instance.Tags, "Name"); tagName != "" {
				name = tagName
			}
			endpoint := strings.Trim(strings.Join([]string{
				aws.StringValue(instance.PrivateIpAddress),
				aws.StringValue(instance.PublicIpAddress),
			}, " / "), " /")

			state := aws.StringValue(instance.State.Name)
			assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
				AccountID:     accountID,
				Region:        region,
				Category:      "compute",
				ResourceType:  "EC2",
				Name:          name,
				Endpoint:      endpoint,
				LastCheckedAt: lastCheckedAt,
				Source:        "aws",
				ExternalID:    "aws:ec2:" + region + ":" + instanceID,
				Status:        normalizeRuntimeStatus(state),
				Tags:          mergeTags([]string{"aws", "ec2", state}, stringifyEC2Tags(instance.Tags)),
				Notes:         fmt.Sprintf("Instance ID: %s; VPC: %s; Subnet: %s", instanceID, aws.StringValue(instance.VpcId), aws.StringValue(instance.SubnetId)),
				Specs: map[string]string{
					"instance_id":   instanceID,
					"instance_type": aws.StringValue(instance.InstanceType),
					"private_ip":    aws.StringValue(instance.PrivateIpAddress),
					"public_ip":     aws.StringValue(instance.PublicIpAddress),
					"vpc_id":        aws.StringValue(instance.VpcId),
					"subnet_id":     aws.StringValue(instance.SubnetId),
					"az":            aws.StringValue(instance.Placement.AvailabilityZone),
					"state":         state,
				},
			}))
		}
	}

	return assets, nil
}

func discoverEBSVolumes(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, volume := range output.Volumes {
		volumeID := aws.StringValue(volume.VolumeId)
		if volumeID == "" {
			continue
		}
		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "storage",
			ResourceType:  "EBS Volume",
			Name:          volumeID,
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:ebs:" + region + ":" + volumeID,
			Status:        normalizeRuntimeStatus(aws.StringValue(volume.State)),
			Tags:          mergeTags([]string{"aws", "ebs"}, stringifyEC2Tags(volume.Tags)),
			Notes:         fmt.Sprintf("SizeGiB: %d; Type: %s; AZ: %s", aws.Int64Value(volume.Size), aws.StringValue(volume.VolumeType), aws.StringValue(volume.AvailabilityZone)),
			Specs: map[string]string{
				"volume_id": volumeID,
				"size_gib":  fmt.Sprintf("%d", aws.Int64Value(volume.Size)),
				"type":      aws.StringValue(volume.VolumeType),
				"az":        aws.StringValue(volume.AvailabilityZone),
				"state":     aws.StringValue(volume.State),
				"encrypted": fmt.Sprintf("%t", aws.BoolValue(volume.Encrypted)),
			},
		}))
	}
	return assets, nil
}

func discoverElasticIPs(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, address := range output.Addresses {
		allocationID := aws.StringValue(address.AllocationId)
		publicIP := aws.StringValue(address.PublicIp)
		if allocationID == "" && publicIP == "" {
			continue
		}
		name := publicIP
		if name == "" {
			name = allocationID
		}
		externalID := allocationID
		if externalID == "" {
			externalID = publicIP
		}

		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "network",
			ResourceType:  "Elastic IP",
			Name:          name,
			Endpoint:      publicIP,
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:eip:" + region + ":" + externalID,
			Status:        "active",
			Tags:          mergeTags([]string{"aws", "eip"}, stringifyEC2Tags(address.Tags)),
			Notes:         fmt.Sprintf("Allocation ID: %s; Association ID: %s", allocationID, aws.StringValue(address.AssociationId)),
			Specs: map[string]string{
				"allocation_id":  allocationID,
				"association_id": aws.StringValue(address.AssociationId),
				"public_ip":      publicIP,
				"private_ip":     aws.StringValue(address.PrivateIpAddress),
				"network_border": aws.StringValue(address.NetworkBorderGroup),
			},
		}))
	}
	return assets, nil
}

func discoverVPCs(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, vpc := range output.Vpcs {
		vpcID := aws.StringValue(vpc.VpcId)
		if vpcID == "" {
			continue
		}
		name := vpcID
		if tagName := findEC2Tag(vpc.Tags, "Name"); tagName != "" {
			name = tagName
		}
		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "network",
			ResourceType:  "VPC",
			Name:          name,
			Endpoint:      aws.StringValue(vpc.CidrBlock),
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:vpc:" + region + ":" + vpcID,
			Status:        normalizeRuntimeStatus(aws.StringValue(vpc.State)),
			Tags:          mergeTags([]string{"aws", "vpc"}, stringifyEC2Tags(vpc.Tags)),
			Notes:         fmt.Sprintf("VPC ID: %s; Tenancy: %s", vpcID, aws.StringValue(vpc.InstanceTenancy)),
			Specs: map[string]string{
				"vpc_id":     vpcID,
				"cidr":       aws.StringValue(vpc.CidrBlock),
				"tenancy":    aws.StringValue(vpc.InstanceTenancy),
				"is_default": fmt.Sprintf("%t", aws.BoolValue(vpc.IsDefault)),
				"state":      aws.StringValue(vpc.State),
			},
		}))
	}
	return assets, nil
}

func discoverSubnets(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, subnet := range output.Subnets {
		subnetID := aws.StringValue(subnet.SubnetId)
		if subnetID == "" {
			continue
		}
		name := subnetID
		if tagName := findEC2Tag(subnet.Tags, "Name"); tagName != "" {
			name = tagName
		}
		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "network",
			ResourceType:  "Subnet",
			Name:          name,
			Endpoint:      aws.StringValue(subnet.CidrBlock),
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:subnet:" + region + ":" + subnetID,
			Status:        "active",
			Tags:          mergeTags([]string{"aws", "subnet"}, stringifyEC2Tags(subnet.Tags)),
			Notes:         fmt.Sprintf("Subnet ID: %s; VPC: %s; AZ: %s", subnetID, aws.StringValue(subnet.VpcId), aws.StringValue(subnet.AvailabilityZone)),
			Specs: map[string]string{
				"subnet_id":          subnetID,
				"vpc_id":             aws.StringValue(subnet.VpcId),
				"cidr":               aws.StringValue(subnet.CidrBlock),
				"az":                 aws.StringValue(subnet.AvailabilityZone),
				"available_ip_count": fmt.Sprintf("%d", aws.Int64Value(subnet.AvailableIpAddressCount)),
				"is_default_for_az":  fmt.Sprintf("%t", aws.BoolValue(subnet.DefaultForAz)),
			},
		}))
	}
	return assets, nil
}

func discoverSecurityGroups(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := ec2.New(sess)
	output, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, sg := range output.SecurityGroups {
		groupID := aws.StringValue(sg.GroupId)
		if groupID == "" {
			continue
		}
		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "network",
			ResourceType:  "Security Group",
			Name:          aws.StringValue(sg.GroupName),
			Endpoint:      groupID,
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:sg:" + region + ":" + groupID,
			Status:        "active",
			Tags:          mergeTags([]string{"aws", "security-group"}, stringifyEC2Tags(sg.Tags)),
			Notes:         fmt.Sprintf("Group ID: %s; VPC: %s", groupID, aws.StringValue(sg.VpcId)),
			Specs: map[string]string{
				"group_id":      groupID,
				"group_name":    aws.StringValue(sg.GroupName),
				"description":   aws.StringValue(sg.Description),
				"vpc_id":        aws.StringValue(sg.VpcId),
				"ingress_rules": fmt.Sprintf("%d", len(sg.IpPermissions)),
				"egress_rules":  fmt.Sprintf("%d", len(sg.IpPermissionsEgress)),
			},
		}))
	}
	return assets, nil
}

func discoverRDSInstances(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := rds.New(sess)
	output, err := client.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}

	var assets []model.Asset
	for _, instance := range output.DBInstances {
		dbID := aws.StringValue(instance.DBInstanceIdentifier)
		if dbID == "" {
			continue
		}
		endpoint := ""
		if instance.Endpoint != nil {
			endpoint = fmt.Sprintf("%s:%d", aws.StringValue(instance.Endpoint.Address), aws.Int64Value(instance.Endpoint.Port))
			endpoint = strings.Trim(endpoint, ":0")
		}
		tagValues := getRDSTags(client, instance.DBInstanceArn)
		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "database",
			ResourceType:  "RDS",
			Name:          dbID,
			Endpoint:      endpoint,
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:rds:" + region + ":" + dbID,
			Status:        normalizeRuntimeStatus(aws.StringValue(instance.DBInstanceStatus)),
			Tags:          mergeTags([]string{"aws", "rds", strings.ToLower(aws.StringValue(instance.Engine))}, tagValues),
			Notes:         fmt.Sprintf("Engine: %s; Class: %s; MultiAZ: %t", aws.StringValue(instance.Engine), aws.StringValue(instance.DBInstanceClass), aws.BoolValue(instance.MultiAZ)),
			Specs: map[string]string{
				"db_instance_id": dbID,
				"engine":         aws.StringValue(instance.Engine),
				"class":          aws.StringValue(instance.DBInstanceClass),
				"endpoint":       endpoint,
				"multi_az":       fmt.Sprintf("%t", aws.BoolValue(instance.MultiAZ)),
				"storage_gib":    fmt.Sprintf("%d", aws.Int64Value(instance.AllocatedStorage)),
				"status":         aws.StringValue(instance.DBInstanceStatus),
			},
		}))
	}
	return assets, nil
}

func discoverLoadBalancers(sess *session.Session, account model.CloudAccount, accountID, region, lastCheckedAt string) ([]model.Asset, error) {
	client := elbv2.New(sess)
	output, err := client.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		if isUnsupportedOperation(err) {
			return nil, nil
		}
		return nil, err
	}

	var assets []model.Asset
	for _, lb := range output.LoadBalancers {
		arn := aws.StringValue(lb.LoadBalancerArn)
		if arn == "" {
			continue
		}
		lbType := aws.StringValue(lb.Type)
		resourceType := "Load Balancer"
		if lbType != "" {
			resourceType = strings.ToUpper(lbType) + " Load Balancer"
		}
		tagValues := getELBV2Tags(client, lb.LoadBalancerArn)

		assets = append(assets, buildDiscoveredAsset(account, discoveredAssetInput{
			AccountID:     accountID,
			Region:        region,
			Category:      "network",
			ResourceType:  resourceType,
			Name:          aws.StringValue(lb.LoadBalancerName),
			Endpoint:      aws.StringValue(lb.DNSName),
			LastCheckedAt: lastCheckedAt,
			Source:        "aws",
			ExternalID:    "aws:elbv2:" + region + ":" + arn,
			Status:        normalizeRuntimeStatus(aws.StringValue(lb.State.Code)),
			Tags:          mergeTags([]string{"aws", "load-balancer", strings.ToLower(lbType)}, tagValues),
			Notes:         fmt.Sprintf("Scheme: %s; VPC: %s", aws.StringValue(lb.Scheme), aws.StringValue(lb.VpcId)),
			Specs: map[string]string{
				"arn":    arn,
				"type":   lbType,
				"dns":    aws.StringValue(lb.DNSName),
				"scheme": aws.StringValue(lb.Scheme),
				"vpc_id": aws.StringValue(lb.VpcId),
				"state":  aws.StringValue(lb.State.Code),
			},
		}))
	}
	return assets, nil
}

type discoveredAssetInput struct {
	AccountID     string
	Region        string
	Category      string
	ResourceType  string
	Name          string
	Endpoint      string
	LastCheckedAt string
	Source        string
	ExternalID    string
	Status        string
	Tags          []string
	Notes         string
	Specs         map[string]string
}

func buildDiscoveredAsset(account model.CloudAccount, input discoveredAssetInput) model.Asset {
	tags := dedupeTags(input.Tags)
	projectCode := inferAWSProjectCode(account, tags, input)
	environment := inferAWSEnvironment(account, tags, input)
	return model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     account.PlatformCode,
		PlatformName:     account.PlatformName,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        input.AccountID,
		ProjectCode:      projectCode,
		Category:         input.Category,
		ResourceType:     input.ResourceType,
		Region:           input.Region,
		Environment:      environment,
		Name:             input.Name,
		Endpoint:         input.Endpoint,
		Owner:            account.Owner,
		Status:           input.Status,
		Criticality:      account.Criticality,
		LastCheckedAt:    input.LastCheckedAt,
		Tags:             tags,
		Notes:            input.Notes,
		Specs:            input.Specs,
		Source:           input.Source,
		ExternalID:       input.ExternalID,
	}
}

func dedupeTags(tags []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

func mergeTags(base []string, extra []string) []string {
	return append(base, extra...)
}

func stringifyEC2Tags(tags []*ec2.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = appendAWSDiscoveryTag(result, aws.StringValue(tag.Key), aws.StringValue(tag.Value))
	}
	return result
}

func stringifyS3Tags(tags []*s3.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = appendAWSDiscoveryTag(result, aws.StringValue(tag.Key), aws.StringValue(tag.Value))
	}
	return result
}

func stringifyRDSTags(tags []*rds.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = appendAWSDiscoveryTag(result, aws.StringValue(tag.Key), aws.StringValue(tag.Value))
	}
	return result
}

func stringifyELBV2Tags(tags []*elbv2.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = appendAWSDiscoveryTag(result, aws.StringValue(tag.Key), aws.StringValue(tag.Value))
	}
	return result
}

func appendAWSDiscoveryTag(tags []string, key, value string) []string {
	key = cleanAWSDiscoveryTagText(key)
	if key == "" {
		return tags
	}
	value = cleanAWSDiscoveryTagText(value)
	if value == "" {
		return append(tags, "tag:"+key)
	}
	return append(tags, "tag:"+key+"="+value)
}

func cleanAWSDiscoveryTagText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func getS3BucketTags(client *s3.S3, bucket string) ([]string, error) {
	output, err := client.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(bucket)})
	if err != nil {
		if isNoTagSet(err) {
			return nil, nil
		}
		return nil, err
	}
	return stringifyS3Tags(output.TagSet), nil
}

func getRDSTags(client *rds.RDS, arn *string) []string {
	if strings.TrimSpace(aws.StringValue(arn)) == "" {
		return nil
	}
	output, err := client.ListTagsForResource(&rds.ListTagsForResourceInput{ResourceName: arn})
	if err != nil {
		return nil
	}
	return stringifyRDSTags(output.TagList)
}

func getELBV2Tags(client *elbv2.ELBV2, arn *string) []string {
	if strings.TrimSpace(aws.StringValue(arn)) == "" {
		return nil
	}
	output, err := client.DescribeTags(&elbv2.DescribeTagsInput{ResourceArns: []*string{arn}})
	if err != nil || len(output.TagDescriptions) == 0 {
		return nil
	}
	return stringifyELBV2Tags(output.TagDescriptions[0].Tags)
}

func findEC2Tag(tags []*ec2.Tag, key string) string {
	for _, tag := range tags {
		if strings.EqualFold(aws.StringValue(tag.Key), key) {
			return aws.StringValue(tag.Value)
		}
	}
	return ""
}

func inferAWSProjectCode(account model.CloudAccount, tags []string, input discoveredAssetInput) string {
	for _, key := range []string{"project", "projectcode", "project-code", "biz", "business", "system", "app"} {
		if value := awsTagValue(tags, key); value != "" {
			if project := normalizeAWSProjectCode(value); project != "" {
				return project
			}
		}
	}

	blob := strings.Join([]string{
		account.Name,
		account.Owner,
		input.Name,
		input.Endpoint,
		input.Notes,
		strings.Join(tags, " "),
	}, " ")
	if project := inferAWSProjectFromText(blob); project != "" {
		return project
	}
	if project := normalizeAWSKnownProjectCode(account.Name); project != "" {
		return project
	}
	return "public"
}

func inferAWSEnvironment(account model.CloudAccount, tags []string, input discoveredAssetInput) string {
	for _, key := range []string{"environment", "env", "stage", "profile"} {
		if value := awsTagValue(tags, key); value != "" {
			if env := normalizeAWSEnvironment(value); env != "" {
				return env
			}
		}
	}

	blob := strings.Join([]string{
		input.Name,
		input.Endpoint,
		input.Notes,
		strings.Join(tags, " "),
	}, " ")
	if env := inferAWSEnvironmentFromText(blob); env != "" {
		return env
	}

	if !isMixedAWSAccount(account) {
		if env := normalizeAWSEnvironment(account.Environment); env != "" {
			return env
		}
	}
	return "unknown"
}

func awsTagValue(tags []string, wantedKey string) string {
	wantedKey = normalizeAWSLookupKey(wantedKey)
	for _, tag := range tags {
		key, value, ok := splitAWSDiscoveryTag(tag)
		if !ok || normalizeAWSLookupKey(key) != wantedKey {
			continue
		}
		if value != "" {
			return value
		}
	}
	return ""
}

func splitAWSDiscoveryTag(tag string) (string, string, bool) {
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(strings.ToLower(tag), "tag:") {
		return "", "", false
	}
	body := strings.TrimSpace(tag[len("tag:"):])
	if body == "" {
		return "", "", false
	}
	key, value, found := strings.Cut(body, "=")
	if !found {
		return strings.TrimSpace(key), "", true
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func normalizeAWSLookupKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "", "/", "")
	return replacer.Replace(value)
}

func normalizeAWSProjectCode(value string) string {
	if known := normalizeAWSKnownProjectCode(value); known != "" {
		return known
	}
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func normalizeAWSKnownProjectCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	switch value {
	case "", "none", "unknown", "n/a", "na":
		return ""
	case "common", "global", "shared", "public-resource", "公共", "公共资源":
		return "public"
	case "pve", "proxmox":
		return "pve"
	case "business":
		return "business"
	case "enterprise", "ent":
		return "enterprise"
	case "cloud":
		return "cloud"
	case "edge":
		return "edge"
	default:
		return ""
	}
}

func inferAWSProjectFromText(value string) string {
	tokens := tokenizeAWSDiscoveryText(value)
	for _, token := range tokens {
		switch token {
		case "pve", "proxmox":
			return "pve"
		case "enterprise", "ent":
			return "enterprise"
		case "edge":
			return "edge"
		case "business":
			return "business"
		case "cloud":
			return "cloud"
		}
	}
	return ""
}

func normalizeAWSEnvironment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	switch value {
	case "", "none", "n/a", "na":
		return ""
	case "dev", "develop", "development":
		return "dev"
	case "test", "testing", "qa":
		return "test"
	case "pre", "preprod", "pre-prod", "staging", "stage", "stg":
		return "staging"
	case "prod", "production":
		return "prod"
	case "local":
		return "local"
	default:
		return value
	}
}

func inferAWSEnvironmentFromText(value string) string {
	tokens := tokenizeAWSDiscoveryText(value)
	for _, token := range tokens {
		if env := normalizeAWSEnvironment(token); env != "" {
			switch env {
			case "dev", "test", "staging", "prod", "local":
				return env
			}
		}
	}
	return ""
}

func tokenizeAWSDiscoveryText(value string) []string {
	value = strings.ToLower(value)
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

func isMixedAWSAccount(account model.CloudAccount) bool {
	value := strings.ToLower(strings.TrimSpace(account.Environment))
	return value == "" || value == "mixed" || value == "unknown"
}

func normalizeRuntimeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "running", "available", "active", "in-use", "inservice":
		return "active"
	case "stopped", "stopping", "pending", "modifying", "backing-up", "creating", "provisioning", "rebooting":
		return "maintenance"
	case "deleted", "deleting", "terminated", "shutting-down", "failed", "inactive":
		return "offline"
	default:
		return "maintenance"
	}
}

func normalizeS3Region(location *string) string {
	value := strings.TrimSpace(aws.StringValue(location))
	switch value {
	case "", "null":
		return defaultAWSRegion
	case "EU":
		return "eu-west-1"
	default:
		return value
	}
}

func isUnsupportedOperation(err error) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		code := awsErr.Code()
		return code == "UnsupportedOperation" || code == "LoadBalancerNotFound"
	}
	return false
}

func isNoTagSet(err error) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		code := awsErr.Code()
		return code == "NoSuchTagSet" || code == "NoSuchTagSetError"
	}
	return false
}
