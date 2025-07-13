import React, { useEffect, useRef, useState, useCallback } from "react";
import type {
  NetworkData,
  NetworkNode,
  NetworkLink,
} from "../types/goldpinger";

// Import the new framework
import { VisualizationFramework } from "./NetworkGraph/core/VisualizationFramework";
import {
  ConfigurationManager,
  type NetworkGraphConfig,
} from "./NetworkGraph/core/ConfigurationManager";
import {
  PluginRegistry,
  type PluginFactory,
} from "./NetworkGraph/core/PluginRegistry";

// Import plugins
import { ForceSimulationPlugin } from "./NetworkGraph/plugins/ForceSimulationPlugin";
import { NodeRenderPlugin } from "./NetworkGraph/plugins/NodeRenderPlugin";
import { EdgeRenderPlugin } from "./NetworkGraph/plugins/EdgeRenderPlugin";
import { InteractionPlugin } from "./NetworkGraph/plugins/InteractionPlugin";
import { LabelsPlugin } from "./NetworkGraph/plugins/LabelsPlugin";

// Register plugins
const registry = PluginRegistry.getInstance();

const forceSimulationFactory: PluginFactory<ForceSimulationPlugin> = {
  create: (config) => new ForceSimulationPlugin(config),
  defaultConfig: {
    enabled: true,
    forces: {
      link: { distance: 360, strength: 0.5 },
      charge: { strength: -200 },
      center: { x: 0, y: 0 },
      collision: { radius: 25 },
    },
  },
  description: "Handles force-directed layout simulation",
};

const nodeRenderFactory: PluginFactory<NodeRenderPlugin> = {
  create: (config) => new NodeRenderPlugin(config),
  defaultConfig: {},
  description: "Renders network nodes as SVG circles",
};

const edgeRenderFactory: PluginFactory<EdgeRenderPlugin> = {
  create: (config) => new EdgeRenderPlugin(config),
  defaultConfig: {
    markers: { enabled: true, type: "arrow" },
    bidirectional: {
      enabled: true,
      mode: "curved",
      curvature: 0.2,
      offset: 5,
    },
  },
  description: "Renders network edges as SVG lines with optional markers",
};

const interactionFactory: PluginFactory<InteractionPlugin> = {
  create: (config) => new InteractionPlugin(config),
  defaultConfig: {
    drag: { enabled: true },
    zoom: { enabled: true, scaleExtent: [0.1, 4] },
    hover: { enabled: true, highlightNeighbors: true },
    click: { enabled: true },
  },
  description: "Handles user interactions like drag, zoom, and click",
};

const labelsFactory: PluginFactory<LabelsPlugin> = {
  create: (config) => new LabelsPlugin(config),
  defaultConfig: {
    enabled: true,
    fontSize: 10,
    fontFamily: "Arial, sans-serif",
    textAnchor: "middle",
    offset: { x: 0, y: 20 },
    color: () => "#333",
    showFor: (node: NetworkNode) => true, // Show labels for all nodes by default
    formatter: (node: NetworkNode) => node.podName || node.label.split("\n")[0], // First line only
  },
  description: "Renders labels for nodes based on their properties",
};

// Register all plugins
registry.register("force-simulation", forceSimulationFactory);
registry.register("node-render", nodeRenderFactory);
registry.register("edge-render", edgeRenderFactory);
registry.register("interaction", interactionFactory);
registry.register("labels", labelsFactory);

interface NetworkGraphProps {
  data: NetworkData;
  width?: number;
  height?: number;
  config?: Partial<NetworkGraphConfig>;
  onNodeClick?: (node: NetworkNode) => void;
  onLinkClick?: (link: NetworkLink) => void;
  onConfigChange?: (config: NetworkGraphConfig) => void;
  className?: string;
}

export const NetworkGraph: React.FC<NetworkGraphProps> = ({
  data,
  width = 800,
  height = 600,
  config: externalConfig,
  onNodeClick,
  onLinkClick,
  onConfigChange,
  className = "",
}) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const frameworkRef = useRef<VisualizationFramework | null>(null);
  const configManagerRef = useRef<ConfigurationManager | null>(null);
  const [hoveredNode, setHoveredNode] = useState<NetworkNode | null>(null);

  // Initialize the framework
  const initializeFramework = useCallback(() => {
    console.log("Initializing NetworkGraph framework...");
    if (!svgRef.current) return;

    // Create configuration manager
    const configManager = new ConfigurationManager({
      width,
      height,
      ...externalConfig,
    });
    configManagerRef.current = configManager;

    // Listen for config changes
    configManager.onConfigChange((newConfig) => {
      onConfigChange?.(newConfig);
    });

    // Create visualization framework
    const framework = new VisualizationFramework(
      svgRef.current,
      configManager.getConfig()
    );
    frameworkRef.current = framework;

    // Add plugins in dependency order
    const pluginsToAdd = [
      "force-simulation",
      "edge-render",
      "node-render",
      "interaction",
      "labels",
    ];

    pluginsToAdd.forEach((pluginName) => {
      if (configManager.isPluginEnabled(pluginName)) {
        try {
          const plugin = registry.create(
            pluginName,
            registry.getDefaultConfig(pluginName)
          );
          framework.addPlugin(plugin);
        } catch (error) {
          console.error(`Failed to add plugin ${pluginName}:`, error);
        }
      }
    });

    // Set up external event handlers
    const context = framework.getContext();

    context.eventBus.on("external:node:click", (node: NetworkNode) => {
      onNodeClick?.(node);
    });

    context.eventBus.on("external:edge:click", (link: NetworkLink) => {
      onLinkClick?.(link);
    });

    context.eventBus.on("external:node:hover", (node: NetworkNode) => {
      setHoveredNode(node);
    });

    context.eventBus.on("external:node:hover:end", () => {
      setHoveredNode(null);
    });

    context.eventBus.on("external:background:click", () => {
      setHoveredNode(null);
      // Could emit a background click event here
    });
  }, [width, height, externalConfig, onNodeClick, onLinkClick, onConfigChange]);

  // Initialize framework on mount
  useEffect(() => {
    console.log("Mounting NetworkGraph component...");
    initializeFramework();

    return () => {
      // Cleanup
      console.log("Unmounting NetworkGraph component...");
      frameworkRef.current?.destroy();
      frameworkRef.current = null;
      configManagerRef.current = null;
    };
  }, []);

  // Update data when it changes
  useEffect(() => {
    console.log("Data is: ");
    console.log("Updating NetworkGraph data...");
    if (frameworkRef.current) {
      frameworkRef.current.updateData(data);
    }
  }, [data]);

  // Update config when external config changes
  useEffect(() => {
    console.log("Updating NetworkGraph config...");
    if (configManagerRef.current && externalConfig) {
      configManagerRef.current.updateConfig(externalConfig);

      // Update framework config
      if (frameworkRef.current) {
        frameworkRef.current.updateConfig(configManagerRef.current.getConfig());
      }
    }
  }, [externalConfig]);

  // Update dimensions
  useEffect(() => {
    console.log("Updating NetworkGraph dimensions...");
    if (configManagerRef.current) {
      configManagerRef.current.setDimensions(width, height);
    }
  }, [width, height]);

  return (
    <div className={`network-graph-v2 relative ${className}`}>
      <svg
        ref={svgRef}
        width={width}
        height={height}
        className="border border-gray-300 rounded-lg bg-white"
        style={{ maxWidth: "100%", height: "auto" }}
      >
        {/* SVG content will be managed by the framework */}
      </svg>

      {/* Tooltip */}
      {hoveredNode && (
        <div className="absolute top-2 left-2 bg-black text-white p-2 rounded text-sm pointer-events-none">
          <div className="font-medium">{hoveredNode.label}</div>
          <div className="text-xs opacity-75">
            {hoveredNode.type} • {hoveredNode.status}
          </div>
        </div>
      )}

      {/* Legend */}
      <div className="absolute top-2 right-2 bg-white p-3 rounded-lg border shadow-sm">
        <div className="text-sm font-semibold mb-2">Legend</div>
        <div className="space-y-1 text-xs">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span>Healthy Pod</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <span>External Target</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-red-500"></div>
            <span>Unhealthy</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-0.5 bg-green-500"></div>
            <span>&lt; 5ms</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-0.5 bg-yellow-500"></div>
            <span>&lt; 50ms</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-0.5 bg-orange-500"></div>
            <span>&gt; 50ms</span>
          </div>
        </div>
      </div>

      {/* Debug info (can be removed in production) */}
      {process.env.NODE_ENV === "development" && (
        <div className="absolute bottom-2 left-2 bg-gray-800 text-white p-2 rounded text-xs">
          <div>Plugins: {registry.getAvailablePlugins().join(", ")}</div>
          <div>Nodes: {data.nodes.length}</div>
          <div>Links: {data.links.length}</div>
        </div>
      )}
    </div>
  );
};
