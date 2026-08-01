import React, { useEffect, useMemo, useState } from "react";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";
import { LevelsApi, UnitsApi } from "../../../helpers/api";

export default function HierarchyTree() {
  const [levels, setLevels] = useState([]);
  const [tree, setTree] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedNode, setSelectedNode] = useState(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [saving, setSaving] = useState(false);
  const [editName, setEditName] = useState("");
  const [childName, setChildName] = useState("");

  useEffect(() => {
    async function load() {
      setLoading(true);
      setError("");
      try {
        const [lvl, { tree: t }] = await Promise.all([
          LevelsApi.list(),
          UnitsApi.tree(),
        ]);
        setLevels(lvl);
        setTree(t || []);
        setSelectedNode(null);
        setEditName("");
        setChildName("");
      } catch (e) {
        setError(e?.response?.data?.error || e.message || "Failed to load tree");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  // Auto-dismiss success messages after 5 seconds
  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => {
        setSuccess("");
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [success]);

  function mapToRcNodes(nodes) {
    return nodes.map((n) => ({
      key: String(n.id),
      title: n.name,
      children: n.children ? mapToRcNodes(n.children) : [],
      raw: n,
    }));
  }

  const rcTreeData = mapToRcNodes(tree);

  // Default: expand top-level territories (children remain collapsed).
  const defaultExpandedKeys = useMemo(() => {
    return rcTreeData.map((node) => node.key);
  }, [rcTreeData]);

  const levelById = new Map(levels.map((l) => [l.id, l]));

  const selectedLevel =
    selectedNode && levelById.get(selectedNode.levelId)
      ? levelById.get(selectedNode.levelId)
      : null;

  const possibleChildLevels = selectedLevel
    ? levels.filter((l) => l.level_number === selectedLevel.level_number + 1)
    : [];

  async function refreshTree(preserveSelection = false, preserveSuccess = false) {
    setLoading(true);
    setError("");
    if (!preserveSuccess) {
      setSuccess("");
    }
    const selectedNodeId = preserveSelection && selectedNode ? selectedNode.id : null;
    try {
      const [lvl, { tree: t }] = await Promise.all([
        LevelsApi.list(),
        UnitsApi.tree(),
      ]);
      setLevels(lvl);
      setTree(t || []);
      
      // Re-select the same node if we're preserving selection
      if (preserveSelection && selectedNodeId) {
        function findNodeById(nodes, targetId) {
          for (const node of nodes) {
            if (node.id === targetId) return node;
            if (node.children) {
              const found = findNodeById(node.children, targetId);
              if (found) return found;
            }
          }
          return null;
        }
        const foundNode = findNodeById(t || [], selectedNodeId);
        if (foundNode) {
          setSelectedNode(foundNode);
          setEditName(foundNode.name || "");
        } else {
          setSelectedNode(null);
          setEditName("");
        }
      } else {
        setSelectedNode(null);
        setEditName("");
      }
      setChildName("");
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load tree");
    } finally {
      setLoading(false);
      setSaving(false);
    }
  }

  async function handleRename(e) {
    e.preventDefault();
    if (!selectedNode || !editName.trim()) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await UnitsApi.update(selectedNode.id, { name: editName.trim() });
      setSuccess("Unit renamed successfully!");
      await refreshTree(true, true); // Preserve selection and success message
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to update unit");
      setSaving(false);
    }
  }

  async function handleAddChild(e) {
    e.preventDefault();
    if (!selectedNode || !childName.trim()) return;

    const selectedLevel =
      selectedNode && levelById.get(selectedNode.levelId)
        ? levelById.get(selectedNode.levelId)
        : null;

    const childLevel = selectedLevel
      ? levels.find((l) => l.level_number === selectedLevel.level_number + 1)
      : null;

    if (!childLevel) {
      setError("No deeper level available for this unit to add a child.");
      setSuccess("");
      return;
    }

    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await UnitsApi.create({
        name: childName.trim(),
        code: undefined,
        levelId: Number(childLevel.id),
        parentId: selectedNode.id,
      });
      setSuccess(`Child unit "${childName.trim()}" added successfully!`);
      setChildName(""); // Clear the input
      await refreshTree(true, true); // Refresh tree but preserve selection and success message
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to create child unit");
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!selectedNode) return;
    const hasChildren = selectedNode.children && selectedNode.children.length > 0;
    const ok = window.confirm(
      hasChildren
        ? "This unit has children. Delete it and its entire subtree?"
        : "Delete this unit?"
    );
    if (!ok) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await UnitsApi.remove(selectedNode.id, { cascade: hasChildren });
      setSuccess("Unit deleted successfully!");
      await refreshTree();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete unit");
      setSaving(false);
    }
  }

  return (
    <div className="card shadow-sm">
      <div className="card-header d-flex justify-content-between align-items-center">
        <div>
          <div className="fw-semibold">Hierarchy Tree</div>
          <div className="text-muted small">
            View the nested structure of all administrative units.
          </div>
        </div>
      </div>
      <div className="card-body">
        {loading && <div className="text-muted small">Loading...</div>}
        {success && (
          <div className="alert alert-success py-1 small mb-3 d-flex justify-content-between align-items-center">
            <span>{success}</span>
            <button
              type="button"
              className="btn-close btn-close-sm"
              onClick={() => setSuccess("")}
              aria-label="Close"
            ></button>
          </div>
        )}
        {error && (
          <div className="alert alert-danger py-1 small mb-3 d-flex justify-content-between align-items-center">
            <span>{error}</span>
            <button
              type="button"
              className="btn-close btn-close-sm"
              onClick={() => setError("")}
              aria-label="Close"
            ></button>
          </div>
        )}

        {!loading && (
          <div className="row">
            <div className="col-md-7 mb-3 mb-md-0">
              <div className="border rounded p-2" style={{ maxHeight: 480, overflow: "auto" }}>
                <Tree
                  treeData={rcTreeData}
                  defaultExpandedKeys={defaultExpandedKeys}
                  onSelect={(_, info) => {
                    const node = info?.node?.raw;
                    setSelectedNode(node || null);
                    setEditName(node?.name || "");
                    setChildName("");
                  }}
                />
              </div>
            </div>
            <div className="col-md-5">
              <div className="border rounded p-3 bg-light h-100">
                <div className="fw-semibold mb-2">Details & actions</div>
                {selectedNode ? (
                  <>
                    <dl className="row small mb-2">
                      <dt className="col-4">MFL UID</dt>
                      <dd className="col-8">
                        <code>{selectedNode.mfl_uid}</code>
                      </dd>

                      <dt className="col-4">Level</dt>
                      <dd className="col-8">
                        {selectedLevel?.name || (
                          <span className="text-muted">Unknown</span>
                        )}
                      </dd>

                      <dt className="col-4">Children</dt>
                      <dd className="col-8">
                        {selectedNode.children?.length || 0}
                      </dd>
                    </dl>

                    <form className="mb-3" onSubmit={handleRename}>
                      <div className="mb-1 small fw-semibold">Rename unit</div>
                      <div className="input-group input-group-sm mb-1">
                        <input
                          className="form-control form-control-sm"
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          placeholder="New name"
                          required
                        />
                        <button
                          type="submit"
                          className="btn btn-primary btn-sm"
                          disabled={saving}
                        >
                          Save
                        </button>
                      </div>
                    </form>

                    <form className="mb-3" onSubmit={handleAddChild}>
                      <div className="mb-1 small fw-semibold">Add child unit</div>
                      <div className="mb-1">
                        <div className="small text-muted mb-1">
                          {possibleChildLevels.length === 0
                            ? "No deeper level available under this unit."
                            : `Child level: ${possibleChildLevels[0].name}`}
                        </div>
                        <input
                          className="form-control form-control-sm mb-1"
                          value={childName}
                          onChange={(e) => setChildName(e.target.value)}
                          placeholder="Child name"
                          disabled={possibleChildLevels.length === 0}
                          required
                        />
                        <button
                          type="submit"
                          className="btn btn-success btn-sm w-100"
                          disabled={saving || possibleChildLevels.length === 0}
                        >
                          Add child
                        </button>
                      </div>
                    </form>

                    <button
                      type="button"
                      className="btn btn-outline-danger btn-sm w-100"
                      onClick={handleDelete}
                      disabled={saving}
                    >
                      Delete unit
                    </button>
                  </>
                ) : (
                  <div className="text-muted small">
                    Select a node in the tree to see its details and actions.
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

