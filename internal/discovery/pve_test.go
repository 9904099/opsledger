package discovery

import (
	"testing"

	"github.com/9904099/opsledger/internal/model"
)

func TestPVEExtendedParsers(t *testing.T) {
	sections := map[string]string{
		"LXC_101": "hostname: ct-one\nmemory: 512\nrootfs: local-lvm:vm-101-disk-0,size=8G",
		"SNAPSHOT_QEMU_100": `[
			{"name":"current"},
			{"name":"before-upgrade","description":"before patch","snaptime":1770000000,"vmstate":true}
		]`,
		"SNAPSHOT_LXC_101": `[
			{"name":"nightly","description":"nightly backup","snaptime":1770000100}
		]`,
	}

	containers := parsePVEContainerList("VMID Status Lock Name\n101 running - ct-one\n", sections)
	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
	if containers[0].VMID != "101" || containers[0].Name != "ct-one" || containers[0].Config["hostname"] != "ct-one" {
		t.Fatalf("unexpected container: %#v", containers[0])
	}

	snapshots := parsePVEGuestSnapshots(sections)
	if len(snapshots) != 2 {
		t.Fatalf("snapshots len = %d, want 2", len(snapshots))
	}
	if snapshots[0].Records[0].Name == "current" {
		t.Fatalf("current snapshot should be filtered: %#v", snapshots[0])
	}

	jobs := parsePVEBackupJobs("0 2 * * * root vzdump 100 --storage local --mode snapshot\n")
	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	if jobs[0].Target != "100" || jobs[0].Storage != "local" || jobs[0].Mode != "snapshot" {
		t.Fatalf("unexpected job: %#v", jobs[0])
	}
}

func TestPVEAssetsFromExtendedSnapshot(t *testing.T) {
	account := model.CloudAccount{
		ID:            "pve-account",
		PlatformCode:  "pve",
		PlatformName:  "PVE",
		Name:          "pve1",
		AccountID:     "198.51.100.36",
		DefaultRegion: "local",
		Environment:   "local",
		Owner:         "Ops",
		Criticality:   "medium",
	}
	snapshot := pveSnapshot{
		Host:         "198.51.100.36",
		Hostname:     "pve1",
		ResourceDate: "2026-06-01",
		Cluster:      []pveClusterItem{{ID: "node/pve1", Name: "pve1", Type: "node", IP: "198.51.100.36", Online: 1}},
		LXCs:         []pveContainer{{VMID: "101", Name: "ct-one", Status: "running"}},
		Pools:        []pvePool{{PoolID: "prod", Comment: "prod pool"}},
		Snapshots:    []pveGuestSnapshot{{GuestType: "qemu", VMID: "100", Name: "qemu-100", Records: []pveSnapshotRecord{{Name: "before-upgrade"}}}},
		BackupJobs:   []pveBackupJob{{ID: "job-1", Target: "100", Storage: "local", Mode: "snapshot"}},
	}

	assets := pveAssetsFromSnapshot(account, snapshot)
	resourceTypes := map[string]int{}
	for _, asset := range assets {
		resourceTypes[asset.ResourceType]++
	}
	for _, resourceType := range []string{"Host", "Cluster Status", "LXC", "Resource Pool", "Snapshot", "Backup Job"} {
		if resourceTypes[resourceType] == 0 {
			t.Fatalf("resource type %s not found in %#v", resourceType, resourceTypes)
		}
	}
}
