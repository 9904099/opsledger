package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

const defaultPVESSHUser = "root"

type PVEImporter struct {
	store store.Store
}

type pveSnapshot struct {
	Host         string
	Hostname     string
	PVEVersion   string
	Nodes        []pveNode
	Cluster      []pveClusterItem
	VMs          []pveVM
	LXCs         []pveContainer
	Storage      []pveStorage
	Bridges      []pveBridge
	Pools        []pvePool
	Snapshots    []pveGuestSnapshot
	BackupJobs   []pveBackupJob
	Warnings     []string
	ResourceDate string
}

type pveNode struct {
	Node    string  `json:"node"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	Mem     int64   `json:"mem"`
	MaxMem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	MaxDisk int64   `json:"maxdisk"`
	Uptime  int64   `json:"uptime"`
}

type pveVM struct {
	VMID       string
	Name       string
	Status     string
	MemoryMB   string
	BootDiskGB string
	PID        string
	Config     map[string]string
}

type pveContainer struct {
	VMID   string
	Name   string
	Status string
	Lock   string
	Config map[string]string
}

type pveStorage struct {
	Name      string
	Type      string
	Status    string
	Total     string
	Used      string
	Available string
	Usage     string
}

type pveBridge struct {
	Name        string
	State       string
	Address     string
	Gateway     string
	BridgePorts string
	STP         string
	FD          string
}

type pveClusterItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	IP      string `json:"ip"`
	NodeID  int    `json:"nodeid"`
	Online  int    `json:"online"`
	Quorate int    `json:"quorate"`
	Local   int    `json:"local"`
}

type pvePool struct {
	PoolID  string `json:"poolid"`
	Comment string `json:"comment"`
}

type pveSnapshotRecord struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      string `json:"parent"`
	SnapTime    int64  `json:"snaptime"`
	VMState     bool   `json:"vmstate"`
}

type pveGuestSnapshot struct {
	GuestType string
	VMID      string
	Name      string
	Records   []pveSnapshotRecord
}

type pveBackupJob struct {
	ID       string
	Schedule string
	Command  string
	Target   string
	Storage  string
	Mode     string
	RawLine  string
}

type pveIPLink struct {
	IfName    string `json:"ifname"`
	Operstate string `json:"operstate"`
	AddrInfo  []struct {
		Local     string `json:"local"`
		PrefixLen int    `json:"prefixlen"`
	} `json:"addr_info"`
}

func NewPVEImporter(dataStore store.Store) *PVEImporter {
	return &PVEImporter{store: dataStore}
}

func (i *PVEImporter) SyncCloudAccount(ctx context.Context, req model.CloudAccountSyncRequest) (model.CloudAccountSyncResult, error) {
	if strings.TrimSpace(req.CloudAccountID) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud_account_id is required")
	}

	account, err := i.store.GetCloudAccount(ctx, req.CloudAccountID)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}
	if strings.ToLower(account.PlatformCode) != "pve" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s is not a PVE account", account.Name)
	}

	host := firstNonEmpty(strings.TrimSpace(account.AccountID), hostFromEndpoint(account.DefaultRegion), account.Name)
	if strings.TrimSpace(host) == "" {
		return model.CloudAccountSyncResult{}, fmt.Errorf("cloud account %s has no PVE host/account_id", account.Name)
	}

	startedAt := time.Now().Format(time.RFC3339)
	snapshot, err := collectPVESnapshot(ctx, account, host)
	if err != nil {
		return model.CloudAccountSyncResult{}, err
	}

	result := model.CloudAccountSyncResult{
		CloudAccountID:    account.ID,
		CloudAccountName:  account.Name,
		PlatformCode:      account.PlatformCode,
		AccountID:         host,
		Regions:           []string{firstNonEmpty(snapshot.Hostname, host)},
		ResourceBreakdown: map[string]int{},
		Warnings:          snapshot.Warnings,
		StartedAt:         startedAt,
		FinishedAt:        time.Now().Format(time.RFC3339),
	}

	assets := pveAssetsFromSnapshot(account, snapshot)
	result.DiscoveredAssets = len(assets)
	for _, asset := range assets {
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

func collectPVESnapshot(ctx context.Context, account model.CloudAccount, host string) (pveSnapshot, error) {
	stdout, stderr, err := runPVESSH(ctx, host, pveDiscoveryScript())
	if err != nil {
		return pveSnapshot{}, fmt.Errorf("pve ssh discovery failed: %w: %s", err, strings.TrimSpace(stderr))
	}

	sections := splitPVESections(stdout)
	snapshot := pveSnapshot{
		Host:         host,
		Hostname:     firstLine(sections["HOSTNAME"]),
		PVEVersion:   firstLine(sections["PVEVERSION"]),
		ResourceDate: time.Now().Format("2006-01-02"),
	}
	if strings.TrimSpace(stderr) != "" {
		snapshot.Warnings = append(snapshot.Warnings, compactWarning(stderr))
	}
	if raw := strings.TrimSpace(sections["NODES"]); raw != "" {
		if err := json.Unmarshal([]byte(raw), &snapshot.Nodes); err != nil {
			snapshot.Warnings = append(snapshot.Warnings, "nodes json: "+err.Error())
		}
	}
	if raw := strings.TrimSpace(sections["CLUSTER_STATUS"]); raw != "" {
		if err := json.Unmarshal([]byte(raw), &snapshot.Cluster); err != nil {
			snapshot.Warnings = append(snapshot.Warnings, "cluster status json: "+err.Error())
		}
	}
	if raw := strings.TrimSpace(sections["POOLS"]); raw != "" {
		if err := json.Unmarshal([]byte(raw), &snapshot.Pools); err != nil {
			snapshot.Warnings = append(snapshot.Warnings, "pools json: "+err.Error())
		}
	}
	snapshot.VMs = parsePVEVMList(sections["QM_LIST"], sections)
	snapshot.LXCs = parsePVEContainerList(sections["LXC_LIST"], sections)
	snapshot.Snapshots = parsePVEGuestSnapshots(sections)
	snapshot.BackupJobs = parsePVEBackupJobs(sections["VZDUMP_CRON"])
	snapshot.Storage = parsePVEStorage(sections["STORAGE"])
	snapshot.Bridges = parsePVEBridges(sections["INTERFACES"], sections["NETWORK"])
	if snapshot.Hostname == "" {
		snapshot.Hostname = account.Name
	}
	return snapshot, nil
}

func runPVESSH(ctx context.Context, host, script string) (string, string, error) {
	keyPath := pveSSHKeyPath()
	args := []string{
		"-i", keyPath,
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=8",
		"-o", "StrictHostKeyChecking=" + pveStrictHostKeyCheckingValue(),
	}
	if knownHosts := pveKnownHostsPath(); knownHosts != "" {
		args = append(args, "-o", "UserKnownHostsFile="+knownHosts)
	}
	args = append(args, defaultPVESSHUser+"@"+host, "bash", "-s")
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = strings.NewReader(script)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func pveSSHKeyPath() string {
	if value := strings.TrimSpace(os.Getenv("OPSLEDGER_PVE_SSH_KEY")); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "~/.ssh/chen-rsa"
	}
	return filepath.Join(home, ".ssh", "chen-rsa")
}

func pveStrictHostKeyCheckingValue() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("OPSLEDGER_PVE_SSH_STRICT_HOST_KEY")))
	if value == "1" || value == "true" || value == "yes" || value == "on" {
		return "yes"
	}
	return "no"
}

func pveKnownHostsPath() string {
	if value := strings.TrimSpace(os.Getenv("OPSLEDGER_PVE_SSH_KNOWN_HOSTS")); value != "" {
		return value
	}
	if pveStrictHostKeyCheckingValue() == "yes" {
		return ""
	}
	return "/tmp/opsledger-pve-known-hosts"
}

func pveDiscoveryScript() string {
	return `
set +e
section() { printf '\n__OPSLEDGER_%s__\n' "$1"; }
section HOSTNAME
hostname
section PVEVERSION
pveversion
section NODES
pvesh get /nodes --output-format json
section CLUSTER_STATUS
pvesh get /cluster/status --output-format json
section POOLS
pvesh get /pools --output-format json
section QM_LIST
qm list
section VM_CONFIGS
for id in $(qm list 2>/dev/null | awk 'NR>1 {print $1}'); do
  printf '\n__OPSLEDGER_VM_%s__\n' "$id"
  qm config "$id"
done
section LXC_LIST
pct list
section LXC_CONFIGS
for id in $(pct list 2>/dev/null | awk 'NR>1 {print $1}'); do
  printf '\n__OPSLEDGER_LXC_%s__\n' "$id"
  pct config "$id"
done
node="$(hostname)"
section SNAPSHOTS
for id in $(qm list 2>/dev/null | awk 'NR>1 {print $1}'); do
  printf '\n__OPSLEDGER_SNAPSHOT_QEMU_%s__\n' "$id"
  pvesh get /nodes/"$node"/qemu/"$id"/snapshot --output-format json
done
for id in $(pct list 2>/dev/null | awk 'NR>1 {print $1}'); do
  printf '\n__OPSLEDGER_SNAPSHOT_LXC_%s__\n' "$id"
  pvesh get /nodes/"$node"/lxc/"$id"/snapshot --output-format json
done
section STORAGE
pvesm status
section VZDUMP_CRON
cat /etc/pve/vzdump.cron
section NETWORK
ip -j -br addr show
section INTERFACES
cat /etc/network/interfaces
`
}

func splitPVESections(output string) map[string]string {
	sections := map[string]string{}
	current := ""
	var builder strings.Builder
	flush := func() {
		if current != "" {
			sections[current] = strings.TrimSpace(builder.String())
			builder.Reset()
		}
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "__OPSLEDGER_") && strings.HasSuffix(line, "__") {
			flush()
			current = strings.TrimSuffix(strings.TrimPrefix(line, "__OPSLEDGER_"), "__")
			continue
		}
		if current != "" {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	flush()
	return sections
}

func parsePVEVMList(raw string, sections map[string]string) []pveVM {
	var vms []pveVM
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "VMID ") || strings.Contains(line, "/var/log/pve") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		// VM names may contain spaces; status/mem/disk/pid are the last four columns.
		vmid := fields[0]
		tail := fields[len(fields)-4:]
		name := strings.Join(fields[1:len(fields)-4], " ")
		vms = append(vms, pveVM{
			VMID:       vmid,
			Name:       name,
			Status:     tail[0],
			MemoryMB:   tail[1],
			BootDiskGB: tail[2],
			PID:        tail[3],
			Config:     parseKeyValueLines(sections["VM_"+vmid]),
		})
	}
	return vms
}

func parsePVEContainerList(raw string, sections map[string]string) []pveContainer {
	var containers []pveContainer
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "VMID ") || strings.Contains(line, "/var/log/pve") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		container := pveContainer{
			VMID:   fields[0],
			Status: fields[1],
			Config: parseKeyValueLines(sections["LXC_"+fields[0]]),
		}
		if len(fields) == 3 {
			container.Name = fields[2]
		} else {
			container.Lock = fields[2]
			container.Name = strings.Join(fields[3:], " ")
		}
		containers = append(containers, container)
	}
	return containers
}

func parsePVEGuestSnapshots(sections map[string]string) []pveGuestSnapshot {
	var snapshots []pveGuestSnapshot
	for section, raw := range sections {
		guestType, vmid, ok := strings.Cut(strings.TrimPrefix(section, "SNAPSHOT_"), "_")
		if !strings.HasPrefix(section, "SNAPSHOT_") || !ok {
			continue
		}
		var records []pveSnapshotRecord
		if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &records); err != nil {
			continue
		}
		filtered := make([]pveSnapshotRecord, 0, len(records))
		for _, record := range records {
			if record.Name == "" || record.Name == "current" {
				continue
			}
			filtered = append(filtered, record)
		}
		if len(filtered) == 0 {
			continue
		}
		snapshots = append(snapshots, pveGuestSnapshot{
			GuestType: strings.ToLower(guestType),
			VMID:      vmid,
			Name:      strings.ToLower(guestType) + "-" + vmid,
			Records:   filtered,
		})
	}
	return snapshots
}

func parsePVEBackupJobs(raw string) []pveBackupJob {
	var jobs []pveBackupJob
	for index, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "vzdump") || strings.Contains(line, "/var/log/pve") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		commandIndex := -1
		for i, field := range fields {
			if field == "vzdump" {
				commandIndex = i
				break
			}
		}
		if commandIndex < 0 || commandIndex+1 >= len(fields) {
			continue
		}
		commandFields := fields[commandIndex+1:]
		jobs = append(jobs, pveBackupJob{
			ID:       fmt.Sprintf("job-%d", index+1),
			Schedule: strings.Join(fields[:5], " "),
			Command:  strings.Join(commandFields, " "),
			Target:   firstBackupCommandValue(commandFields),
			Storage:  backupOptionValue(commandFields, "--storage"),
			Mode:     backupOptionValue(commandFields, "--mode"),
			RawLine:  line,
		})
	}
	return jobs
}

func firstBackupCommandValue(fields []string) string {
	for _, field := range fields {
		if !strings.HasPrefix(field, "-") {
			return field
		}
	}
	return ""
}

func backupOptionValue(fields []string, option string) string {
	for index, field := range fields {
		if field == option && index+1 < len(fields) {
			return fields[index+1]
		}
	}
	return ""
}

func parsePVEStorage(raw string) []pveStorage {
	var items []pveStorage
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name ") || strings.Contains(line, "/var/log/pve") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		items = append(items, pveStorage{
			Name:      fields[0],
			Type:      fields[1],
			Status:    fields[2],
			Total:     fields[3],
			Used:      fields[4],
			Available: fields[5],
			Usage:     fields[6],
		})
	}
	return items
}

func parsePVEBridges(interfacesRaw, networkRaw string) []pveBridge {
	stateByName := map[string]string{}
	addressByName := map[string]string{}
	var links []pveIPLink
	if err := json.Unmarshal([]byte(strings.TrimSpace(networkRaw)), &links); err == nil {
		for _, link := range links {
			stateByName[link.IfName] = link.Operstate
			if len(link.AddrInfo) > 0 {
				addressByName[link.IfName] = fmt.Sprintf("%s/%d", link.AddrInfo[0].Local, link.AddrInfo[0].PrefixLen)
			}
		}
	}

	var bridges []pveBridge
	var current *pveBridge
	flush := func() {
		if current == nil {
			return
		}
		if current.Name != "" && (strings.HasPrefix(current.Name, "vmbr") || current.BridgePorts != "") {
			if current.State == "" {
				current.State = stateByName[current.Name]
			}
			if current.Address == "" {
				current.Address = addressByName[current.Name]
			}
			bridges = append(bridges, *current)
		}
		current = nil
	}
	for _, line := range strings.Split(interfacesRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && fields[0] == "iface" {
			flush()
			current = &pveBridge{Name: fields[1]}
			continue
		}
		if current == nil || len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "address":
			current.Address = fields[1]
		case "gateway":
			current.Gateway = fields[1]
		case "bridge-ports":
			current.BridgePorts = strings.Join(fields[1:], " ")
		case "bridge-stp":
			current.STP = fields[1]
		case "bridge-fd":
			current.FD = fields[1]
		}
	}
	flush()
	return bridges
}

func parseKeyValueLines(raw string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "/var/log/pve") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result
}

func pveAssetsFromSnapshot(account model.CloudAccount, snapshot pveSnapshot) []model.Asset {
	today := snapshot.ResourceDate
	var assets []model.Asset
	nodeName := firstNonEmpty(snapshot.Hostname, account.Name)
	assets = append(assets, model.Asset{
		PlatformID:       account.PlatformID,
		PlatformCode:     "pve",
		PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
		Category:         "compute",
		ResourceType:     "Host",
		Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
		Environment:      account.Environment,
		Name:             nodeName,
		Endpoint:         "https://" + snapshot.Host + ":8006",
		Owner:            account.Owner,
		Status:           "active",
		Criticality:      account.Criticality,
		LastCheckedAt:    today,
		Tags:             []string{"pve", "proxmox", "host"},
		Notes:            "PVE 宿主节点，使用默认 SSH 密钥自动发现",
		Specs:            pveHostSpecs(snapshot),
		Source:           "pve",
		ExternalID:       fmt.Sprintf("pve:%s:host:%s", account.ID, nodeName),
	})

	for _, vm := range snapshot.VMs {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "compute",
			ResourceType:     "VM",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             firstNonEmpty(vm.Name, "vm-"+vm.VMID),
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           pveVMStatus(vm.Status),
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "vm", strings.ToLower(vm.Status)},
			Notes:            "PVE VM 自动发现",
			Specs:            pveVMSpecs(vm),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:vm:%s", account.ID, vm.VMID),
		})
	}

	for _, container := range snapshot.LXCs {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "compute",
			ResourceType:     "LXC",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             firstNonEmpty(container.Name, "lxc-"+container.VMID),
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           pveVMStatus(container.Status),
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "lxc", strings.ToLower(container.Status)},
			Notes:            "PVE LXC 自动发现",
			Specs:            pveContainerSpecs(container),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:lxc:%s", account.ID, container.VMID),
		})
	}

	for _, storage := range snapshot.Storage {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "storage",
			ResourceType:     "Storage",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             storage.Name,
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           pveActiveStatus(storage.Status),
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "storage", storage.Type},
			Notes:            "PVE 存储自动发现",
			Specs:            pveStorageSpecs(storage),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:storage:%s", account.ID, storage.Name),
		})
	}

	for _, cluster := range snapshot.Cluster {
		clusterID := firstNonEmpty(cluster.ID, cluster.Name)
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "compute",
			ResourceType:     "Cluster Status",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             clusterID,
			Endpoint:         firstNonEmpty(cluster.IP, snapshot.Host),
			Owner:            account.Owner,
			Status:           pveClusterStatus(cluster),
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "cluster", cluster.Type},
			Notes:            "PVE 集群状态自动发现",
			Specs:            pveClusterSpecs(cluster),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:cluster:%s", account.ID, clusterID),
		})
	}

	for _, pool := range snapshot.Pools {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "compute",
			ResourceType:     "Resource Pool",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             pool.PoolID,
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           "active",
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "resource-pool"},
			Notes:            "PVE 资源池自动发现",
			Specs:            pvePoolSpecs(pool),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:pool:%s", account.ID, pool.PoolID),
		})
	}

	for _, snapshotGroup := range snapshot.Snapshots {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "storage",
			ResourceType:     "Snapshot",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             snapshotGroup.Name,
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           "active",
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "snapshot", snapshotGroup.GuestType},
			Notes:            "PVE 快照自动发现",
			Specs:            pveSnapshotSpecs(snapshotGroup),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:snapshot:%s:%s", account.ID, snapshotGroup.GuestType, snapshotGroup.VMID),
		})
	}

	for _, job := range snapshot.BackupJobs {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "storage",
			ResourceType:     "Backup Job",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             firstNonEmpty(job.Target, job.ID),
			Endpoint:         "https://" + snapshot.Host + ":8006",
			Owner:            account.Owner,
			Status:           "active",
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "backup", "vzdump"},
			Notes:            "PVE 备份任务自动发现",
			Specs:            pveBackupJobSpecs(job),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:backup:%s", account.ID, job.ID),
		})
	}

	for _, bridge := range snapshot.Bridges {
		assets = append(assets, model.Asset{
			PlatformID:       account.PlatformID,
			PlatformCode:     "pve",
			PlatformName:     firstNonEmpty(account.PlatformName, "PVE"),
			CloudAccountID:   account.ID,
			CloudAccountName: account.Name,
			AccountID:        firstNonEmpty(account.AccountID, snapshot.Host),
			Category:         "network",
			ResourceType:     "Network Bridge",
			Region:           firstNonEmpty(account.DefaultRegion, "local-lan"),
			Environment:      account.Environment,
			Name:             bridge.Name,
			Endpoint:         firstNonEmpty(bridge.Address, snapshot.Host),
			Owner:            account.Owner,
			Status:           pveBridgeStatus(bridge.State),
			Criticality:      account.Criticality,
			LastCheckedAt:    today,
			Tags:             []string{"pve", "network", "bridge"},
			Notes:            "PVE 网络桥接自动发现",
			Specs:            pveBridgeSpecs(bridge),
			Source:           "pve",
			ExternalID:       fmt.Sprintf("pve:%s:bridge:%s", account.ID, bridge.Name),
		})
	}
	return assets
}

func pveHostSpecs(snapshot pveSnapshot) map[string]string {
	specs := map[string]string{
		"host":        snapshot.Host,
		"hostname":    snapshot.Hostname,
		"pve_version": snapshot.PVEVersion,
	}
	for _, node := range snapshot.Nodes {
		if node.Node == snapshot.Hostname || len(snapshot.Nodes) == 1 {
			specs["node_status"] = node.Status
			specs["cpu_ratio"] = fmt.Sprintf("%.4f", node.CPU)
			specs["memory_used_bytes"] = strconv.FormatInt(node.Mem, 10)
			specs["memory_total_bytes"] = strconv.FormatInt(node.MaxMem, 10)
			specs["disk_used_bytes"] = strconv.FormatInt(node.Disk, 10)
			specs["disk_total_bytes"] = strconv.FormatInt(node.MaxDisk, 10)
			specs["uptime_seconds"] = strconv.FormatInt(node.Uptime, 10)
			break
		}
	}
	return specs
}

func pveVMSpecs(vm pveVM) map[string]string {
	specs := map[string]string{
		"vmid":        vm.VMID,
		"status":      vm.Status,
		"memory_mb":   vm.MemoryMB,
		"bootdisk_gb": vm.BootDiskGB,
		"pid":         vm.PID,
	}
	for key, value := range vm.Config {
		specs["config_"+key] = value
	}
	return specs
}

func pveContainerSpecs(container pveContainer) map[string]string {
	specs := map[string]string{
		"vmid":   container.VMID,
		"status": container.Status,
		"lock":   container.Lock,
	}
	for key, value := range container.Config {
		specs["config_"+key] = value
	}
	return specs
}

func pveClusterSpecs(cluster pveClusterItem) map[string]string {
	return map[string]string{
		"id":      cluster.ID,
		"name":    cluster.Name,
		"type":    cluster.Type,
		"ip":      cluster.IP,
		"node_id": strconv.Itoa(cluster.NodeID),
		"online":  strconv.Itoa(cluster.Online),
		"quorate": strconv.Itoa(cluster.Quorate),
		"local":   strconv.Itoa(cluster.Local),
	}
}

func pvePoolSpecs(pool pvePool) map[string]string {
	return map[string]string{
		"pool_id": pool.PoolID,
		"comment": pool.Comment,
	}
}

func pveSnapshotSpecs(snapshot pveGuestSnapshot) map[string]string {
	specs := map[string]string{
		"guest_type":     snapshot.GuestType,
		"vmid":           snapshot.VMID,
		"snapshot_count": strconv.Itoa(len(snapshot.Records)),
	}
	for index, record := range snapshot.Records {
		prefix := fmt.Sprintf("snapshot_%d_", index+1)
		specs[prefix+"name"] = record.Name
		specs[prefix+"description"] = record.Description
		specs[prefix+"parent"] = record.Parent
		specs[prefix+"snaptime"] = strconv.FormatInt(record.SnapTime, 10)
		specs[prefix+"vmstate"] = fmt.Sprintf("%t", record.VMState)
	}
	return specs
}

func pveBackupJobSpecs(job pveBackupJob) map[string]string {
	return map[string]string{
		"id":       job.ID,
		"schedule": job.Schedule,
		"command":  job.Command,
		"target":   job.Target,
		"storage":  job.Storage,
		"mode":     job.Mode,
		"raw_line": job.RawLine,
	}
}

func pveStorageSpecs(storage pveStorage) map[string]string {
	return map[string]string{
		"type":      storage.Type,
		"status":    storage.Status,
		"total":     storage.Total,
		"used":      storage.Used,
		"available": storage.Available,
		"usage":     storage.Usage,
	}
}

func pveBridgeSpecs(bridge pveBridge) map[string]string {
	return map[string]string{
		"state":        bridge.State,
		"address":      bridge.Address,
		"gateway":      bridge.Gateway,
		"bridge_ports": bridge.BridgePorts,
		"bridge_stp":   bridge.STP,
		"bridge_fd":    bridge.FD,
	}
}

func pveClusterStatus(cluster pveClusterItem) string {
	if cluster.Type == "node" && cluster.Online == 0 {
		return "offline"
	}
	if cluster.Type == "cluster" && cluster.Quorate == 0 {
		return "maintenance"
	}
	return "active"
}

func pveVMStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return "active"
	case "stopped":
		return "offline"
	default:
		return "maintenance"
	}
}

func pveActiveStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "active") {
		return "active"
	}
	if strings.EqualFold(strings.TrimSpace(status), "disabled") {
		return "offline"
	}
	return "maintenance"
}

func pveBridgeStatus(state string) string {
	if strings.EqualFold(strings.TrimSpace(state), "up") {
		return "active"
	}
	if strings.TrimSpace(state) == "" {
		return "maintenance"
	}
	return "offline"
}

func hostFromEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		return ""
	}
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return host
	}
	return value
}

func firstLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "/var/log/pve") {
			return line
		}
	}
	return ""
}

func compactWarning(raw string) string {
	lines := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) >= 3 {
			break
		}
	}
	return strings.Join(lines, " | ")
}
