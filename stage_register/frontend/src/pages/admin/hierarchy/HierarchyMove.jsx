import React, { useEffect, useMemo, useState } from "react";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";
import { UnitsApi } from "../../../helpers/api";

export default function HierarchyMove() {
  const [tree, setTree] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [selectedUnitKeys, setSelectedUnitKeys] = useState([]);
  const [selectedParentNode, setSelectedParentNode] = useState(null);
  const [leftSearchQuery, setLeftSearchQuery] = useState("");
  const [rightSearchQuery, setRightSearchQuery] = useState("");
  const [leftExpandedKeys, setLeftExpandedKeys] = useState([]);
  const [rightExpandedKeys, setRightExpandedKeys] = useState([]);

  useEffect(() => {
    async function load() {
      setLoading(true);
      setError("");
      try {
        const { tree: t } = await UnitsApi.tree();
        setTree(t || []);
        setSelectedUnitKeys([]);
        setSelectedParentNode(null);
      } catch (e) {
        setError(e?.response?.data?.error || e.message || "Failed to load tree");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  function mapToRcNodes(nodes, searchQuery = "") {
    return nodes
      .map((n) => {
        const node = {
          key: String(n.id),
          title: n.name,
          children: n.children ? mapToRcNodes(n.children, searchQuery) : [],
          raw: n,
        };
        // Filter nodes based on search query
        if (searchQuery) {
          const matchesSearch =
            n.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            (node.children && node.children.length > 0);
          if (!matchesSearch && !node.children.length) {
            return null;
          }
        }
        return node;
      })
      .filter(Boolean);
  }

  const rcTreeData = useMemo(() => mapToRcNodes(tree), [tree]);
  const leftTreeData = useMemo(
    () => mapToRcNodes(tree, leftSearchQuery),
    [tree, leftSearchQuery]
  );
  const rightTreeData = useMemo(
    () => mapToRcNodes(tree, rightSearchQuery),
    [tree, rightSearchQuery]
  );

  // Get all selected unit IDs and their descendants (to exclude from right tree)
  const excludedKeys = useMemo(() => {
    const excluded = new Set(selectedUnitKeys);
    const selectedIds = selectedUnitKeys.map((k) => parseInt(k));

    function collectDescendants(node) {
      if (selectedIds.includes(node.id)) {
        function addChildren(n) {
          if (n.children) {
            n.children.forEach((child) => {
              excluded.add(String(child.id));
              addChildren(child);
            });
          }
        }
        addChildren(node);
      }
      if (node.children) {
        node.children.forEach(collectDescendants);
      }
    }

    tree.forEach(collectDescendants);
    return Array.from(excluded);
  }, [selectedUnitKeys, tree]);

  // Clear selected parent if it becomes excluded
  useEffect(() => {
    if (selectedParentNode && excludedKeys.includes(selectedParentNode.key)) {
      setSelectedParentNode(null);
    }
  }, [excludedKeys, selectedParentNode]);

  // Filter right tree to exclude selected units and their descendants
  function filterExcludedNodes(nodes) {
    return nodes
      .map((n) => {
        if (excludedKeys.includes(n.key)) {
          return null;
        }
        return {
          ...n,
          children: n.children ? filterExcludedNodes(n.children) : [],
        };
      })
      .filter(Boolean);
  }

  const filteredRightTreeData = useMemo(
    () => filterExcludedNodes(rightTreeData),
    [rightTreeData, excludedKeys]
  );

  // Default: expand root level
  const defaultExpandedKeys = useMemo(() => {
    if (rcTreeData.length === 0) return [];
    return [rcTreeData[0]?.key].filter(Boolean);
  }, [rcTreeData]);

  // Get selected unit names for display
  const selectedUnitNames = useMemo(() => {
    const names = [];
    function findNodes(nodes, keys) {
      nodes.forEach((n) => {
        if (keys.includes(String(n.id))) {
          names.push(n.name);
        }
        if (n.children) {
          findNodes(n.children, keys);
        }
      });
    }
    findNodes(tree, selectedUnitKeys);
    return names;
  }, [tree, selectedUnitKeys]);

  async function refreshTree() {
    setLoading(true);
    setError("");
    try {
      const { tree: t } = await UnitsApi.tree();
      setTree(t || []);
      setSelectedUnitKeys([]);
      setSelectedParentNode(null);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load tree");
    } finally {
      setLoading(false);
      setSaving(false);
    }
  }

  async function handleMove() {
    if (selectedUnitKeys.length === 0) {
      setError("Please select at least one unit to move.");
      return;
    }
    if (!selectedParentNode) {
      setError("Please select a new parent unit.");
      return;
    }

    const newParentId = selectedParentNode.raw?.id || null;

    setSaving(true);
    setError("");

    try {
      // Move each selected unit
      const movePromises = selectedUnitKeys.map((key) => {
        const unitId = parseInt(key);
        return UnitsApi.move(unitId, newParentId);
      });

      await Promise.all(movePromises);
      await refreshTree();
    } catch (e) {
      setError(
        e?.response?.data?.error || e.message || "Failed to move units"
      );
      setSaving(false);
    }
  }

  function handleLeftCheck(checkedKeys, info) {
    // When checkStrictly is true, checkedKeys is an object with checked and halfChecked
    // We only want the actually checked nodes, not their parents or children
    let keys = [];
    
    if (Array.isArray(checkedKeys)) {
      // If it's an array, use it directly (but this shouldn't happen with checkStrictly)
      keys = checkedKeys;
    } else if (checkedKeys && checkedKeys.checked) {
      // With checkStrictly, we get { checked: [...], halfChecked: [...] }
      keys = checkedKeys.checked || [];
    } else {
      keys = [];
    }
    
    setSelectedUnitKeys(keys);
    
    // Clear selected parent if it becomes invalid (is a selected unit or descendant)
    if (selectedParentNode) {
      const parentKey = selectedParentNode.key;
      if (keys.includes(parentKey)) {
        setSelectedParentNode(null);
      }
    }
  }

  function handleRightSelect(selectedKeys, info) {
    if (info.selected && info.node) {
      setSelectedParentNode(info.node);
    } else {
      setSelectedParentNode(null);
    }
  }

  return (
    <div className="card shadow-sm">
      <div className="card-header">
        <div className="fw-semibold" style={{ fontSize: "1.5rem" }}>
          Hierarchy operations
        </div>
      </div>
      <div className="card-body">
        {loading && <div className="text-muted small">Loading...</div>}
        {error && (
          <div className="alert alert-danger py-1 small mb-3">{error}</div>
        )}

        {!loading && !error && (
          <>
            <div className="row mb-3">
              {/* Left Panel - Source Tree */}
              <div className="col-md-6 mb-3 mb-md-0">
                <div className="mb-2">
                  <input
                    type="text"
                    className="form-control form-control-sm"
                    placeholder="Search"
                    value={leftSearchQuery}
                    onChange={(e) => setLeftSearchQuery(e.target.value)}
                  />
                </div>
                <div
                  className="border rounded p-2"
                  style={{ maxHeight: 480, overflow: "auto" }}
                >
                  <Tree
                    treeData={leftTreeData}
                    checkable
                    checkStrictly
                    checkedKeys={{ checked: selectedUnitKeys, halfChecked: [] }}
                    onCheck={handleLeftCheck}
                    defaultExpandedKeys={defaultExpandedKeys}
                    expandedKeys={leftExpandedKeys}
                    onExpand={setLeftExpandedKeys}
                  />
                </div>
              </div>

              {/* Right Panel - Destination Tree */}
              <div className="col-md-6">
                <div className="mb-2">
                  <input
                    type="text"
                    className="form-control form-control-sm"
                    placeholder="Search"
                    value={rightSearchQuery}
                    onChange={(e) => setRightSearchQuery(e.target.value)}
                  />
                </div>
                <div
                  className="border rounded p-2"
                  style={{ maxHeight: 480, overflow: "auto" }}
                >
                  <Tree
                    treeData={filteredRightTreeData}
                    defaultExpandedKeys={defaultExpandedKeys}
                    expandedKeys={rightExpandedKeys}
                    onExpand={setRightExpandedKeys}
                    onSelect={handleRightSelect}
                    selectedKeys={
                      selectedParentNode ? [selectedParentNode.key] : []
                    }
                    titleRender={(nodeData) => {
                      const isSelected =
                        selectedParentNode?.key === nodeData.key;
                      return (
                        <span
                          style={{
                            color: isSelected ? "#ff8c00" : undefined,
                            fontWeight: isSelected ? "bold" : undefined,
                          }}
                        >
                          {nodeData.title}
                        </span>
                      );
                    }}
                  />
                </div>
              </div>
            </div>

            {/* Bottom Section - Move Controls */}
            <div className="row mt-3">
              <div className="col-md-6">
                <div className="border rounded p-3 bg-light">
                  <div className="fw-semibold mb-2">Move</div>
                  <div className="small text-muted mb-2">
                    Select organisation unit(s) to move from the left.
                  </div>
                  {selectedUnitNames.length > 0 && (
                    <div className="small">
                      <strong>Selected units:</strong>
                      <ul className="mb-0 mt-1">
                        {selectedUnitNames.map((name, idx) => (
                          <li key={idx}>{name}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </div>
              <div className="col-md-6">
                <div className="border rounded p-3 bg-light">
                  <div className="fw-semibold mb-2">New parent</div>
                  {selectedParentNode ? (
                    <div className="small">
                      <ul className="mb-0">
                        <li>{selectedParentNode.raw?.name || selectedParentNode.title}</li>
                      </ul>
                    </div>
                  ) : (
                    <div className="small text-muted">
                      Select a new parent from the right.
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Move Button */}
            <div className="text-center mt-4">
              <button
                type="button"
                className="btn btn-secondary btn-lg"
                onClick={handleMove}
                disabled={
                  saving ||
                  selectedUnitKeys.length === 0 ||
                  !selectedParentNode
                }
                style={{
                  minWidth: "250px",
                  backgroundColor: "#e9ecef",
                  borderColor: "#dee2e6",
                  color: "#212529",
                }}
              >
                {saving ? "Moving..." : "MOVE ORGANISATION UNITS"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

