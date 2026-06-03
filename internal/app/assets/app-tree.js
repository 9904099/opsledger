function groupAssetsByProjectHierarchy(assets) {
  return assets.reduce((groups, asset) => {
    const project = asset.project_code || "public";
    const platform = asset.platform_name || asset.platform_code || "Unknown";
    const account = asset.cloud_account_name || asset.account_id || "Unassigned";
    const resourceType = asset.resource_type || "Unknown";
    if (!groups[project]) {
      groups[project] = {};
    }
    if (!groups[project][platform]) {
      groups[project][platform] = {};
    }
    if (!groups[project][platform][account]) {
      groups[project][platform][account] = {};
    }
    if (!groups[project][platform][account][resourceType]) {
      groups[project][platform][account][resourceType] = [];
    }
    groups[project][platform][account][resourceType].push(asset);
    return groups;
  }, {});
}

function groupAssetsByAccountHierarchy(assets) {
  return assets.reduce((groups, asset) => {
    const platform = asset.platform_name || asset.platform_code || "Unknown";
    const account = asset.cloud_account_name || asset.account_id || "Unassigned";
    const project = asset.project_code || "public";
    const resourceType = asset.resource_type || "Unknown";
    if (!groups[platform]) {
      groups[platform] = {};
    }
    if (!groups[platform][account]) {
      groups[platform][account] = {};
    }
    if (!groups[platform][account][project]) {
      groups[platform][account][project] = {};
    }
    if (!groups[platform][account][project][resourceType]) {
      groups[platform][account][project][resourceType] = [];
    }
    groups[platform][account][project][resourceType].push(asset);
    return groups;
  }, {});
}

function countNestedAssets(node) {
  if (Array.isArray(node)) {
    return node.length;
  }
  return Object.values(node).reduce((sum, child) => sum + countNestedAssets(child), 0);
}

function summarizeNode(node) {
  const items = flattenNode(node);
  return items.reduce((summary, asset) => {
    const status = asset.status || "maintenance";
    summary[status] = (summary[status] || 0) + 1;
    return summary;
  }, {active: 0, maintenance: 0, offline: 0});
}

function flattenNode(node) {
  if (Array.isArray(node)) {
    return node;
  }
  return Object.values(node).flatMap((child) => flattenNode(child));
}

function isTreeKeyExpanded(key, defaultExpanded = false) {
  if (state.collapsedTreeKeys.has(key)) {
    return false;
  }
  if (state.expandedTreeKeys.has(key)) {
    return true;
  }
  return defaultExpanded;
}
