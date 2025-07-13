import * as d3 from "d3";
import type { NetworkData, NetworkNode } from "../../../types/goldpinger";
import type { VisualizationPlugin, VisualizationContext } from "../types";

export interface LabelsPluginConfig {
  enabled: boolean;
  fontSize: number;
  fontFamily: string;
  textAnchor: "start" | "middle" | "end";
  offset: { x: number; y: number };
  color: (node: NetworkNode) => string;
  showFor: (node: NetworkNode) => boolean;
  formatter: (node: NetworkNode) => string;
}

export class LabelsPlugin implements VisualizationPlugin<LabelsPluginConfig> {
  name = "labels";
  dependencies = ["node-render"]; // Needs nodes to be rendered first

  private config: LabelsPluginConfig;
  private labelSelection?: d3.Selection<
    SVGTextElement,
    NetworkNode,
    SVGGElement,
    unknown
  >;

  constructor(config?: Partial<LabelsPluginConfig>) {
    this.config = {
      enabled: true,
      fontSize: 10,
      fontFamily: "Arial, sans-serif",
      textAnchor: "middle",
      offset: { x: 0, y: 20 },
      color: () => "#333",
      showFor: (node) => node.type === "pod", // Only show labels for pods by default
      formatter: (node) => node.podName || node.label.split("\n")[0], // First line only
      ...config,
    };
  }

  init(context: VisualizationContext): void {
    if (!this.config.enabled) return;

    // Listen for simulation ticks to update positions
    context.eventBus.on("simulation:tick", () => {
      this.updatePositions();
    });

    // Listen for node hover to highlight labels
    context.eventBus.on("node:hover", (event: any) => {
      this.highlightLabel(event.node.id);
    });

    context.eventBus.on("node:hover:end", () => {
      this.unhighlightAllLabels();
    });
  }

  update(data: NetworkData, context: VisualizationContext): void {
    if (!this.config.enabled) return;

    const labelsLayer = context.getOrCreateLayer("labels", 3);

    // Filter nodes that should have labels
    const nodesWithLabels = data.nodes.filter(this.config.showFor);

    // Bind data and create/update/remove labels
    this.labelSelection = labelsLayer
      .selectAll<SVGTextElement, NetworkNode>("text")
      .data(nodesWithLabels, (d) => d.id);

    // Remove old labels
    this.labelSelection
      .exit()
      .transition()
      .duration(context.config.animation.duration)
      .style("opacity", 0)
      .remove();

    // Add new labels
    const enterLabels = this.labelSelection
      .enter()
      .append("text")
      .style("opacity", 0)
      .attr("class", "node-label")
      .style("pointer-events", "none")
      .style("user-select", "none");

    // Merge enter and update selections
    this.labelSelection = enterLabels.merge(this.labelSelection);

    // Apply styles to all labels
    this.labelSelection
      .transition()
      .duration(context.config.animation.duration)
      .text(this.config.formatter)
      .attr("font-size", `${this.config.fontSize}px`)
      .attr("font-family", this.config.fontFamily)
      .attr("text-anchor", this.config.textAnchor)
      .attr("fill", this.config.color)
      .style("opacity", 1);

    // Update positions immediately
    this.updatePositions();
  }

  private updatePositions(): void {
    if (!this.labelSelection) return;

    this.labelSelection
      .attr("x", (d) => (d.x || 0) + this.config.offset.x)
      .attr("y", (d) => (d.y || 0) + this.config.offset.y);
  }

  private highlightLabel(nodeId: string): void {
    if (!this.labelSelection) return;

    this.labelSelection
      .filter((d) => d.id === nodeId)
      .transition()
      .duration(150)
      .attr("font-size", `${this.config.fontSize * 1.2}px`)
      .attr("fill", "#ff6b35")
      .style("font-weight", "bold");
  }

  private unhighlightAllLabels(): void {
    if (!this.labelSelection) return;

    this.labelSelection
      .transition()
      .duration(150)
      .attr("font-size", `${this.config.fontSize}px`)
      .attr("fill", this.config.color)
      .style("font-weight", "normal");
  }

  destroy(context: VisualizationContext): void {
    const labelsLayer = context.layers.get("labels");
    if (labelsLayer) {
      labelsLayer.selectAll("text").remove();
    }
    this.labelSelection = undefined;
  }

  getConfig(): LabelsPluginConfig {
    return this.config;
  }

  updateConfig(newConfig: Partial<LabelsPluginConfig>): void {
    this.config = { ...this.config, ...newConfig };

    // Re-apply styles if labels exist
    if (this.labelSelection) {
      this.labelSelection
        .text(this.config.formatter)
        .attr("font-size", `${this.config.fontSize}px`)
        .attr("font-family", this.config.fontFamily)
        .attr("text-anchor", this.config.textAnchor)
        .attr("fill", this.config.color);

      this.updatePositions();
    }
  }

  // Utility methods
  showLabels(): void {
    this.config.enabled = true;
  }

  hideLabels(): void {
    this.config.enabled = false;
    if (this.labelSelection) {
      this.labelSelection.style("opacity", 0);
    }
  }

  setLabelFormat(formatter: (node: NetworkNode) => string): void {
    this.config.formatter = formatter;
    if (this.labelSelection) {
      this.labelSelection.text(this.config.formatter);
    }
  }
}
