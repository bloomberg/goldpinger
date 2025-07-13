import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { NetworkGraph } from "../components/NetworkGraph";
import { transformGoldpingerData } from "../utils/networkData";
import { useGoldpingerData } from "../hooks/useGoldpingerData";
import { getNodeLabel, validateLink } from "../utils/networkHelpers";
import type { NetworkNode, NetworkLink } from "../types/goldpinger";

export const Route = createFileRoute("/network")({
  component: NetworkPage,
});

function NetworkPage() {
  const [selectedNode, setSelectedNode] = useState<NetworkNode | null>(null);
  const [selectedLink, setSelectedLink] = useState<NetworkLink | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);

  const { data, loading, error, refresh, lastUpdated } = useGoldpingerData({
    autoRefresh,
    refreshInterval: 10000, // 10 seconds when auto-refresh is enabled
  });

  // TODO: REMOVE THIS WHEN DATA IS NOT STATIC
  const networkData = useMemo(
    () => (data ? transformGoldpingerData(data) : null),
    [data]
  );

  const handleNodeClick = (node: NetworkNode) => {
    setSelectedNode(node);
    setSelectedLink(null);
  };

  const handleLinkClick = (link: NetworkLink) => {
    setSelectedLink(link);
    setSelectedNode(null);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-gray-900"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-red-600 text-center">
          <h2 className="text-xl font-semibold mb-2">Error</h2>
          <p>{error}</p>
        </div>
      </div>
    );
  }

  if (!networkData) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-gray-600 text-center">
          <h2 className="text-xl font-semibold mb-2">No Data</h2>
          <p>No network data available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="mb-4">
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              Network Topology
            </h1>
            <p className="text-gray-600">
              Interactive visualization of Goldpinger pod connectivity
            </p>
          </div>
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
            <div className="text-sm text-gray-500">
              {lastUpdated && (
                <span>Last updated: {lastUpdated.toLocaleTimeString()}</span>
              )}
            </div>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">Auto refresh (10s)</span>
            </label>
            <button
              onClick={refresh}
              disabled={loading}
              className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 text-sm"
            >
              {loading ? "Loading..." : "Refresh"}
            </button>
          </div>
        </div>
      </div>

      <div className="flex flex-col lg:flex-row gap-4">
        {/* Main graph area */}
        <div className="flex-1 min-h-0">
          <NetworkGraph
            data={networkData}
            width={1500}
            height={900}
            onNodeClick={handleNodeClick}
            onLinkClick={handleLinkClick}
            className="w-full"
          />
        </div>

        {/* Side panel for details */}
        <div className="w-full lg:w-80 bg-gray-50 rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-3">Details</h3>

          {selectedNode && (
            <div className="space-y-3">
              <h4 className="font-medium text-blue-600">
                Node: {selectedNode.label}
              </h4>
              <div className="bg-white p-3 rounded border">
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="font-medium">Type:</span>{" "}
                    {selectedNode.type}
                  </div>
                  <div>
                    <span className="font-medium">Status:</span>
                    <span
                      className={`ml-1 px-2 py-0.5 rounded text-xs ${
                        selectedNode.status === "healthy"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {selectedNode.status}
                    </span>
                  </div>
                  {selectedNode.podIP && (
                    <div>
                      <span className="font-medium">Pod IP:</span>{" "}
                      {selectedNode.podIP}
                    </div>
                  )}
                  {selectedNode.hostIP && (
                    <div>
                      <span className="font-medium">Host IP:</span>{" "}
                      {selectedNode.hostIP}
                    </div>
                  )}
                </div>
              </div>
              <details className="bg-white rounded border">
                <summary className="p-2 cursor-pointer font-medium">
                  Raw Data
                </summary>
                <pre className="p-2 text-xs bg-gray-100 overflow-auto max-h-32">
                  {JSON.stringify(selectedNode.data, null, 2)}
                </pre>
              </details>
            </div>
          )}

          {selectedLink && (
            <div className="space-y-3">
              <h4 className="font-medium text-blue-600">Connection(s)</h4>
              <div className="bg-white p-3 rounded border">
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="font-medium">From:</span>{" "}
                    {getNodeLabel(selectedLink.source)}
                  </div>
                  <div>
                    <span className="font-medium">To:</span>{" "}
                    {getNodeLabel(selectedLink.target)}
                  </div>
                  <div>
                    <span className="font-medium">Type:</span>{" "}
                    {selectedLink.type}
                  </div>
                  <div>
                    <span className="font-medium">Latency:</span>{" "}
                    {selectedLink.latency}ms
                  </div>
                  <div>
                    <span className="font-medium">Status:</span>
                    <span
                      className={`ml-1 px-2 py-0.5 rounded text-xs ${
                        selectedLink.status === "healthy"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {selectedLink.status}
                    </span>
                  </div>
                </div>
              </div>
              <details className="bg-white rounded border">
                <summary className="p-2 cursor-pointer font-medium">
                  Raw Data
                </summary>
                <pre className="p-2 text-xs bg-gray-100 overflow-auto max-h-48">
                  {JSON.stringify(selectedLink.data, null, 2)}
                </pre>
              </details>
              {selectedLink.reverseLink && (
                <div className="bg-white p-3 rounded border">
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="font-medium">From:</span>{" "}
                      {getNodeLabel(selectedLink.target)}
                    </div>
                    <div>
                      <span className="font-medium">To:</span>{" "}
                      {getNodeLabel(selectedLink.source)}
                    </div>
                    <div>
                      <span className="font-medium">Type:</span>{" "}
                      {selectedLink.reverseLink.type}
                    </div>
                    <div>
                      <span className="font-medium">Latency:</span>{" "}
                      {selectedLink.reverseLink.latency}ms
                    </div>
                    <div>
                      <span className="font-medium">Status:</span>
                      <span
                        className={`ml-1 px-2 py-0.5 rounded text-xs ${
                          selectedLink.reverseLink.status === "healthy"
                            ? "bg-green-100 text-green-800"
                            : "bg-red-100 text-red-800"
                        }`}
                      >
                        {selectedLink.reverseLink.status}
                      </span>
                    </div>
                  </div>
                </div>
              )}
              {selectedLink.reverseLink && (
                <details className="bg-white rounded border">
                  <summary className="p-2 cursor-pointer font-medium">
                    Raw Data
                  </summary>
                  <pre className="p-2 text-xs bg-gray-100 overflow-auto max-h-48">
                    {JSON.stringify(selectedLink.reverseLink.data, null, 2)}
                  </pre>
                </details>
              )}
            </div>
          )}

          {!selectedNode && !selectedLink && (
            <div className="text-gray-500 text-sm">
              Click on a node or connection to view details
            </div>
          )}

          {/* Network Statistics */}
          <div className="mt-6 pt-4 border-t">
            <h4 className="font-medium mb-2">Network Stats</h4>
            <div className="space-y-1 text-sm">
              <div>
                Pods: {networkData.nodes.filter((n) => n.type === "pod").length}
              </div>
              <div>
                External Targets:{" "}
                {networkData.nodes.filter((n) => n.type === "external").length}
              </div>
              <div>Connections: {networkData.links.length}</div>
              <div>
                Healthy:{" "}
                {networkData.links.filter((l) => l.status === "healthy").length}
              </div>
              <div>
                Unhealthy:{" "}
                {
                  networkData.links.filter((l) => l.status === "unhealthy")
                    .length
                }
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
