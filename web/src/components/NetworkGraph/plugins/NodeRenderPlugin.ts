import * as d3 from "d3";
import type { NetworkData, NetworkNode } from "../../../types/goldpinger";
import type {
  VisualizationPlugin,
  VisualizationContext,
  NodePluginConfig,
  NodeEvent,
} from "../types";

export class NodeRenderPlugin implements VisualizationPlugin<NodePluginConfig> {
  name = "node-render";
  dependencies = ["force-simulation"]; // Depends on simulation for positioning

  private config: NodePluginConfig;
  private nodeSelection?: d3.Selection<
    SVGCircleElement,
    NetworkNode,
    SVGGElement,
    unknown
  >;

  constructor(config?: Partial<NodePluginConfig>) {
    this.config = {
      radius: (node: NetworkNode) => (node.type === "pod" ? 12 : 8),
      color: (node: NetworkNode) => {
        switch (node.status) {
          case "healthy":
            return node.type === "pod" ? "#10b981" : "#3b82f6";
          case "unhealthy":
            return "#ef4444";
          default:
            return "#6b7280";
        }
      },
      stroke: () => "#fff",
      strokeWidth: () => 2,
      opacity: () => 1,
      className: (node: NetworkNode) =>
        `network-node node-${node.type} node-${node.status}`,
      ...config,
    };
  }

  init(context: VisualizationContext): void {
    // Listen for simulation ticks to update positions
    context.eventBus.on("simulation:tick", () => {
      this.updatePositions();
    });
  }

  update(data: NetworkData, context: VisualizationContext): void {
    const nodesLayer = context.layers.get("nodes");
    if (!nodesLayer) {
      console.warn("Nodes layer not found");
      return;
    }

    // Bind data and create/update/remove nodes
    this.nodeSelection = nodesLayer
      .selectAll<SVGCircleElement, NetworkNode>("circle")
      .data(data.nodes, (d) => d.id);

    // Remove old nodes
    this.nodeSelection
      .exit()
      .transition()
      .duration(context.config.animation.duration)
      .attr("r", 0)
      .style("opacity", 0)
      .remove();

    // Add new nodes
    const enterNodes = this.nodeSelection
      .enter()
      .append("circle")
      .attr("r", 0)
      .style("opacity", 0)
      .attr("class", this.config.className)
      .style("cursor", "pointer");

    // Merge enter and update selections
    this.nodeSelection = enterNodes.merge(this.nodeSelection);

    // Apply styles to all nodes
    this.nodeSelection
      .transition()
      .duration(context.config.animation.duration)
      .attr("r", this.config.radius)
      .attr("fill", this.config.color)
      .attr("stroke", this.config.stroke)
      .attr("stroke-width", this.config.strokeWidth)
      .style("opacity", this.config.opacity);

    // Set up event handlers
    this.setupEventHandlers(context);
  }

  private setupEventHandlers(context: VisualizationContext): void {
    if (!this.nodeSelection) return;

    this.nodeSelection
      .on("click", (event, node) => {
        event.stopPropagation();
        const nodeEvent: NodeEvent = {
          type: "click",
          node,
          event,
        };
        context.eventBus.emit("node:click", nodeEvent);
      })
      .on("mouseenter", (event, node) => {
        // Apply hover effect
        d3.select(event.target)
          .transition()
          .duration(150)
          .attr("r", this.config.radius(node) * 1.2)
          .attr("stroke-width", this.config.strokeWidth(node) * 1.5);

        const nodeEvent: NodeEvent = {
          type: "hover",
          node,
          event,
        };
        context.eventBus.emit("node:hover", nodeEvent);
      })
      .on("mouseleave", (event, node) => {
        // Remove hover effect
        d3.select(event.target)
          .transition()
          .duration(150)
          .attr("r", this.config.radius(node))
          .attr("stroke-width", this.config.strokeWidth(node));

        context.eventBus.emit("node:hover:end", { node, event });
      });
  }

  private updatePositions(): void {
    if (!this.nodeSelection) return;

    this.nodeSelection.attr("cx", (d) => d.x || 0).attr("cy", (d) => d.y || 0);
  }

  destroy(context: VisualizationContext): void {
    const nodesLayer = context.layers.get("nodes");
    if (nodesLayer) {
      nodesLayer.selectAll("circle").remove();
    }
    this.nodeSelection = undefined;
  }

  getConfig(): NodePluginConfig {
    return this.config;
  }

  updateConfig(newConfig: Partial<NodePluginConfig>): void {
    this.config = { ...this.config, ...newConfig };

    // Re-apply styles if nodes exist
    if (this.nodeSelection) {
      this.nodeSelection
        .attr("r", this.config.radius)
        .attr("fill", this.config.color)
        .attr("stroke", this.config.stroke)
        .attr("stroke-width", this.config.strokeWidth)
        .style("opacity", this.config.opacity)
        .attr("class", this.config.className);
    }
  }

  // Utility methods
  highlightNode(nodeId: string): void {
    if (!this.nodeSelection) return;

    this.nodeSelection
      .filter((d) => d.id === nodeId)
      .transition()
      .duration(200)
      .attr("stroke", "#ff6b35")
      .attr("stroke-width", 4);
  }

  unhighlightNode(nodeId: string): void {
    if (!this.nodeSelection) return;

    this.nodeSelection
      .filter((d) => d.id === nodeId)
      .transition()
      .duration(200)
      .attr("stroke", this.config.stroke)
      .attr("stroke-width", this.config.strokeWidth);
  }

  getNodePosition(nodeId: string): { x: number; y: number } | null {
    if (!this.nodeSelection) return null;

    const node = this.nodeSelection.data().find((d) => d.id === nodeId);
    return node ? { x: node.x || 0, y: node.y || 0 } : null;
  }
}
