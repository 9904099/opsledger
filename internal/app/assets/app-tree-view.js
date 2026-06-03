// Asset tree DOM rendering and Cloudflare hierarchy helpers.

function renderAssetTree() {
  const assets = filterAssets();
  if (!assets.length) {
    refs.assetTree.innerHTML = `<div class="empty">没有匹配的资产记录。</div>`;
    applyI18n();
    return;
  }

  refs.treeModeAccountButton.classList.toggle("active", state.treeMode === "account");
  refs.treeModeProjectButton.classList.toggle("active", state.treeMode === "project");
  refs.assetTree.innerHTML = state.treeMode === "project" ? renderProjectModeTree(assets) : renderAccountModeTree(assets);
  bindAssetTreeEvents();
  applyI18n();
}

function renderProjectModeTree(assets) {
  const grouped = groupAssetsByProjectHierarchy(assets);
  const projects = Object.keys(grouped).sort((a, b) => projectLabel(a).localeCompare(projectLabel(b)));

  return projects.map((project) => {
    const projectKey = `project:${project}`;
    const projectExpanded = isTreeKeyExpanded(projectKey, true);
    const platforms = Object.keys(grouped[project]).sort();
    return `
      <section class="tree-group tree-level-1">
        ${renderTreeToggle(projectKey, "项目", projectLabel(project), countNestedAssets(grouped[project]), projectExpanded, summarizeNode(grouped[project]))}
        <div class="tree-group-items ${projectExpanded ? "" : "hidden"}">
          ${platforms.map((platform) => {
            const platformKey = `${projectKey}/platform:${platform}`;
            const platformExpanded = isTreeKeyExpanded(platformKey, true);
            const accounts = Object.keys(grouped[project][platform]).sort();
            return `
              <section class="tree-group tree-level-2">
                ${renderTreeToggle(platformKey, "平台", platform, countNestedAssets(grouped[project][platform]), platformExpanded, summarizeNode(grouped[project][platform]))}
                <div class="tree-group-items ${platformExpanded ? "" : "hidden"}">
                  ${accounts.map((account) => {
                    const accountKey = `${platformKey}/account:${account}`;
                    const accountNode = grouped[project][platform][account];
                    const accountIsCloudflare = isCloudflarePlatform(platform, accountNode);
                    const accountExpanded = isTreeKeyExpanded(accountKey, accountIsCloudflare);
                    const cloudAccount = findCloudAccountByName(account);
                    const accountSelectAttrs = cloudAccount ? `data-cloud-account-select-id="${escapeHTML(cloudAccount.id)}"` : "";
                    return `
                      <section class="tree-group tree-level-3">
                        ${renderTreeToggle(accountKey, "账号", account, countNestedAssets(grouped[project][platform][account]), accountExpanded, summarizeNode(grouped[project][platform][account]), renderAccountCurrentCost(cloudAccount), accountSelectAttrs)}
                        <div class="tree-group-items ${accountExpanded ? "" : "hidden"}">
                          ${accountIsCloudflare
                            ? renderCloudflareAccountTree(accountNode, accountKey)
                            : renderResourceTypeTree(accountNode, accountKey)}
                        </div>
                      </section>
                    `;
                  }).join("")}
                </div>
              </section>
            `;
          }).join("")}
        </div>
      </section>
    `;
  }).join("");
}

function renderAccountModeTree(assets) {
  const grouped = groupAssetsByAccountHierarchy(assets);
  const platforms = Object.keys(grouped).sort();

  return platforms.map((platform) => {
    const platformKey = `platform:${platform}`;
    const platformExpanded = isTreeKeyExpanded(platformKey, true);
    const accounts = Object.keys(grouped[platform]).sort();
    return `
      <section class="tree-group tree-level-1">
        ${renderTreeToggle(platformKey, "平台", platform, countNestedAssets(grouped[platform]), platformExpanded, summarizeNode(grouped[platform]))}
        <div class="tree-group-items ${platformExpanded ? "" : "hidden"}">
          ${accounts.map((account) => {
            const accountKey = `${platformKey}/account:${account}`;
            const accountNode = grouped[platform][account];
            const accountExpanded = isTreeKeyExpanded(accountKey, isCloudflarePlatform(platform, accountNode));
            const cloudAccount = findCloudAccountByName(account);
            const accountSelectAttrs = cloudAccount ? `data-cloud-account-select-id="${escapeHTML(cloudAccount.id)}"` : "";
            const projects = Object.keys(accountNode).sort((a, b) => projectLabel(a).localeCompare(projectLabel(b)));
            return `
              <section class="tree-group tree-level-2">
                ${renderTreeToggle(accountKey, "账号", account, countNestedAssets(accountNode), accountExpanded, summarizeNode(accountNode), renderAccountCurrentCost(cloudAccount), accountSelectAttrs)}
                <div class="tree-group-items ${accountExpanded ? "" : "hidden"}">
                  ${projects.map((project) => {
                    const projectKey = `${accountKey}/project:${project}`;
                    const projectNode = accountNode[project];
                    const projectExpanded = isTreeKeyExpanded(projectKey, isCloudflarePlatform(platform, projectNode));
                    return `
                      <section class="tree-group tree-level-3">
                        ${renderTreeToggle(projectKey, "项目", projectLabel(project), countNestedAssets(projectNode), projectExpanded, summarizeNode(projectNode))}
                        <div class="tree-group-items ${projectExpanded ? "" : "hidden"}">
                          ${isCloudflarePlatform(platform, projectNode)
                            ? renderCloudflareAccountTree(projectNode, projectKey)
                            : renderResourceTypeTree(projectNode, projectKey)}
                        </div>
                      </section>
                    `;
                  }).join("")}
                </div>
              </section>
            `;
          }).join("")}
        </div>
      </section>
    `;
  }).join("");
}

function bindAssetTreeEvents() {
  refs.assetTree.querySelectorAll("[data-tree-key]").forEach((button) => {
    button.addEventListener("click", () => {
      const key = button.dataset.treeKey;
      const expanded = button.getAttribute("aria-expanded") === "true";
      if (expanded) {
        state.expandedTreeKeys.delete(key);
        state.collapsedTreeKeys.add(key);
      } else {
        state.expandedTreeKeys.add(key);
        state.collapsedTreeKeys.delete(key);
      }
      if (button.dataset.cloudAccountSelectId) {
        state.selectedCloudAccountId = button.dataset.cloudAccountSelectId;
        state.selectedAssetId = "";
        state.selectedChangeId = "";
        state.detailMode = "cloudAccount";
        syncFormsFromSelection();
        render();
        return;
      }
      renderAssetTree();
    });
  });

  refs.assetTree.querySelectorAll("[data-asset-id]").forEach((button) => {
    button.addEventListener("click", () => {
      state.selectedAssetId = button.dataset.assetId;
      state.selectedChangeId = "";
      state.detailMode = "asset";
      state.assetDetailTab = "overview";
      state.page = "workbench";
      syncFormsFromSelection();
      render();
    });
  });
}

function findCloudAccountByName(name) {
  return state.cloudAccounts.find((item) => item.name === name) || null;
}

function renderAccountCurrentCost(account) {
  if (!account || !account.current_month_cost) {
    return "";
  }
  return `<span class="tree-cost">本月当前使用 ${escapeHTML(`${account.cost_currency || ""} ${account.current_month_cost}`.trim())}</span>`;
}

function renderResourceTypeTree(accountNode, accountKey) {
  return Object.keys(accountNode).sort().map((resourceType) => {
    const typeKey = `${accountKey}/type:${resourceType}`;
    const typeExpanded = isTreeKeyExpanded(typeKey, false);
    const items = accountNode[resourceType];
    return `
      <section class="tree-group tree-level-4">
        ${renderTreeToggle(typeKey, "类型", resourceType, items.length, typeExpanded, summarizeNode(items))}
        <div class="tree-group-items ${typeExpanded ? "" : "hidden"}">
          ${items.map((asset) => renderAssetTreeItem(asset)).join("")}
        </div>
      </section>
    `;
  }).join("");
}

function renderCloudflareAccountTree(accountNode, accountKey) {
  const assets = flattenNode(accountNode);
  const zones = buildCloudflareZoneGroups(assets);
  return zones.map((zone) => {
    const zoneKey = `${accountKey}/zone:${zone.name}`;
    const zoneExpanded = isTreeKeyExpanded(zoneKey, false);
    return `
      <section class="tree-group tree-level-4 tree-zone">
        ${renderTreeToggle(zoneKey, "Zone", zone.name, zone.assets.length, zoneExpanded, summarizeNode(zone.assets), renderZoneExpiry(zone.zoneAsset))}
        <div class="tree-group-items ${zoneExpanded ? "" : "hidden"}">
          ${renderCloudflareTypeGroups(zone, zoneKey)}
        </div>
      </section>
    `;
  }).join("");
}

function buildCloudflareZoneGroups(assets) {
  const zoneNames = [...new Set(assets
    .map((asset) => getCloudflareZoneName(asset))
    .filter(Boolean))]
    .sort(compareDomainNames);

  const zones = zoneNames.map((zoneName) => {
    const zoneAssets = assets.filter((asset) => getCloudflareZoneName(asset) === zoneName);
    const zoneAsset = zoneAssets.find((asset) => asset.resource_type === "Zone" && asset.name === zoneName) || findCloudflareZoneAsset(zoneName);
    const dnsRecords = zoneAssets
      .filter((asset) => asset.resource_type !== "Zone")
      .sort((left, right) => compareDomainNames(left.name || "", right.name || ""));
    const zone = {
      name: zoneName,
      assets: zoneAssets,
      zoneAsset,
      typeGroups: buildCloudflareTypeGroups(zoneName, zoneAsset, dnsRecords)
    };
    return zone;
  });

  const orphanAssets = assets.filter((asset) => !getCloudflareZoneName(asset));
  if (orphanAssets.length) {
    const orphanZone = {
      name: "未归属 Zone",
      assets: orphanAssets,
      zoneAsset: null,
      typeGroups: buildCloudflareTypeGroups("", null, orphanAssets)
    };
    zones.push(orphanZone);
  }

  return zones;
}

function findCloudflareZoneAsset(zoneName) {
  const normalized = String(zoneName || "").toLowerCase();
  if (!normalized) {
    return null;
  }
  return (state.assets || []).find((asset) => (
    asset.platform_code === "cloudflare"
    && asset.resource_type === "Zone"
    && String(asset.name || "").toLowerCase() === normalized
  )) || null;
}

function buildCloudflareTypeGroups(zoneName, zoneAsset, dnsRecords) {
  const groups = new Map();
  if (zoneAsset) {
    groups.set("Zone", {
      label: "Zone",
      assets: [zoneAsset],
      root: createDomainNode("Zone")
    });
    groups.get("Zone").root.assets.push(zoneAsset);
  }

  dnsRecords.forEach((asset) => {
    const recordType = getCloudflareRecordType(asset);
    if (!groups.has(recordType)) {
      groups.set(recordType, {
        label: recordType,
        assets: [],
        root: createDomainNode(recordType)
      });
    }
    const group = groups.get(recordType);
    group.assets.push(asset);
    addCloudflareRecordToZone(group.root, zoneName, asset);
  });

  return [...groups.values()].sort((left, right) => compareRecordTypes(left.label, right.label));
}

function renderCloudflareTypeGroups(zone, zoneKey) {
  return zone.typeGroups.map((group) => {
    const typeKey = `${zoneKey}/record-type:${group.label}`;
    const typeExpanded = isTreeKeyExpanded(typeKey, false);
    return `
      <section class="tree-group tree-level-5 tree-record-type">
        ${renderTreeToggle(typeKey, "类型", group.label, group.assets.length, typeExpanded, summarizeNode(group.assets))}
        <div class="tree-group-items ${typeExpanded ? "" : "hidden"}">
          ${renderDomainBranch(group.root, typeKey, 6)}
        </div>
      </section>
    `;
  }).join("");
}

function renderZoneExpiry(zoneAsset) {
  if (!zoneAsset || !zoneAsset.specs || !zoneAsset.specs.expires_at) {
    return `<span class="zone-expiry unknown">到期未知</span>`;
  }
  const days = Number(zoneAsset.specs.expires_in_days || 0);
  const level = days <= 30 ? "danger" : days <= 90 ? "warn" : "ok";
  return `<span class="zone-expiry ${level}">${escapeHTML(zoneAsset.specs.expires_at)} / ${days}天</span>`;
}

function createDomainNode(label) {
  return {
    label,
    assets: [],
    children: new Map()
  };
}

function addCloudflareRecordToZone(root, zoneName, asset) {
  const labels = getDomainLabelsForZone(asset.name || "", zoneName).slice(0, 5);
  let node = root;
  labels.forEach((label) => {
    if (!node.children.has(label)) {
      node.children.set(label, createDomainNode(label));
    }
    node = node.children.get(label);
  });
  node.assets.push(asset);
}

function renderDomainBranch(node, parentKey, depth) {
  const childSections = [...node.children.values()]
    .sort((left, right) => compareDomainNames(left.label, right.label))
    .map((child) => renderDomainNode(child, parentKey, depth));
  const recordItems = node.assets.map((asset) => renderAssetTreeItem(asset, "dns-record"));
  return [...childSections, ...recordItems].join("");
}

function renderDomainNode(node, parentKey, depth) {
  const nodeKey = `${parentKey}/domain:${node.label}`;
  const assets = flattenDomainNode(node);
  const expanded = isTreeKeyExpanded(nodeKey, depth >= 2);
  const level = Math.min(7, Math.max(5, 10 - depth));
  return `
    <section class="tree-group tree-level-${level} tree-domain">
      ${renderTreeToggle(nodeKey, "域名", node.label, assets.length, expanded, summarizeNode(assets))}
      <div class="tree-group-items ${expanded ? "" : "hidden"}">
        ${renderDomainBranch(node, nodeKey, depth - 1)}
      </div>
    </section>
  `;
}

function flattenDomainNode(node) {
  return [
    ...node.assets,
    ...[...node.children.values()].flatMap((child) => flattenDomainNode(child))
  ];
}

function getCloudflareZoneName(asset) {
  return (asset.specs && asset.specs.zone) || (asset.resource_type === "Zone" ? asset.name : "");
}

function getCloudflareRecordType(asset) {
  return (asset.specs && asset.specs.type) || asset.resource_type || "Unknown";
}

function getDomainLabelsForZone(name, zoneName) {
  const cleanName = String(name || "").toLowerCase().replace(/\.$/, "");
  const cleanZone = String(zoneName || "").toLowerCase().replace(/\.$/, "");
  if (!cleanName) {
    return ["未命名记录"];
  }
  if (!cleanZone || cleanName === cleanZone) {
    return ["@"];
  }
  const suffix = `.${cleanZone}`;
  const relativeName = cleanName.endsWith(suffix) ? cleanName.slice(0, -suffix.length) : cleanName;
  return relativeName.split(".").filter(Boolean).reverse();
}

function compareDomainNames(left, right) {
  return String(left).localeCompare(String(right), "zh-Hans-CN", {numeric: true, sensitivity: "base"});
}

function compareRecordTypes(left, right) {
  const order = ["Zone", "A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA", "HTTPS", "SVCB"];
  const leftIndex = order.indexOf(left);
  const rightIndex = order.indexOf(right);
  if (leftIndex !== -1 || rightIndex !== -1) {
    return (leftIndex === -1 ? order.length : leftIndex) - (rightIndex === -1 ? order.length : rightIndex);
  }
  return left.localeCompare(right, "zh-Hans-CN", {numeric: true, sensitivity: "base"});
}

function isCloudflarePlatform(platform, accountNode) {
  return platform.toLowerCase().includes("cloudflare") || flattenNode(accountNode).some((asset) => asset.platform_code === "cloudflare");
}

function renderAssetTreeItem(asset, variant = "") {
  const meta = asset.platform_code === "cloudflare"
    ? `${escapeHTML((asset.specs && asset.specs.type) || asset.resource_type || "-")} / ${escapeHTML(asset.endpoint || "-")}`
    : `${escapeHTML(projectLabel(asset.project_code))} / ${escapeHTML(asset.region || "-")} / ${escapeHTML(asset.status)}`;
  return `
    <button type="button" class="tree-item ${variant ? `tree-item-${variant}` : ""} ${asset.id === state.selectedAssetId ? "active" : ""} ${asset.status !== "active" ? "risk" : ""} ${asset.criticality === "high" ? "critical" : ""}" data-asset-id="${asset.id}">
      <span class="tree-item-title">${escapeHTML(asset.name)}</span>
      <span class="tree-item-meta">${meta}</span>
    </button>
  `;
}

function renderTreeToggle(key, kind, label, count, expanded, summary, extra = "", attrs = "") {
  return `
    <button class="tree-group-toggle" type="button" data-tree-key="${escapeHTML(key)}" aria-expanded="${expanded}" ${attrs}>
      <span class="tree-group-main">
        <span class="tree-group-arrow">${expanded ? "−" : "+"}</span>
        <span class="tree-kind">${escapeHTML(kind)}</span>
        <span>${escapeHTML(label)}</span>
      </span>
      <span class="tree-group-side">
        ${extra}
        <small>${count}</small>
        <span class="tree-status-summary">
          ${summary.active ? `<span class="tree-status-dot ok" title="active">${summary.active}</span>` : ""}
          ${summary.maintenance ? `<span class="tree-status-dot warn" title="maintenance">${summary.maintenance}</span>` : ""}
          ${summary.offline ? `<span class="tree-status-dot danger" title="offline">${summary.offline}</span>` : ""}
        </span>
      </span>
    </button>
  `;
}
