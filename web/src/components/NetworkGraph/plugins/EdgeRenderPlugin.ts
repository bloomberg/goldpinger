import * as d3 from "d3";
import type { NetworkData, NetworkLink } from "../../../types/goldpinger";
import type {
  VisualizationPlugin,
  VisualizationContext,
  EdgePluginConfig,
  EdgeEvent,
} from "../types";
import { getNodeId } from "../../../utils/networkHelpers";

export class EdgeRenderPlugin implements VisualizationPlugin<EdgePluginConfig> {
  name = "edge-render";
  dependencies = ["force-simulation"]; // Depends on simulation for positioning

  private config: EdgePluginConfig;
  private edgeSelection?: d3.Selection<
    SVGLineElement | SVGPathElement,
    NetworkLink,
    SVGGElement,
    unknown
  >;

  constructor(config?: Partial<EdgePluginConfig>) {
    this.config = {
      width: (edge: NetworkLink) => {
        let width = 2;
        if (edge.latency < 5) width = 3;
        else if (edge.latency > 100) width = 1;
        return width;
      },
      color: (edge: NetworkLink) => {
        const unhealthyEdge =
          edge.status === "unhealthy" ||
          edge.reverseLink?.status === "unhealthy";

        const maxLatency = Math.max(
          edge.latency,
          edge.reverseLink?.latency || 0
        );

        if (unhealthyEdge) return "#ef4444";
        if (maxLatency < 5) return "#10b981";
        else if (maxLatency < 50) return "#f59e0b";
        else return "#f97316";
      },
      opacity: () => 1,
      className: (edge: NetworkLink) => {
        let classes = "network-link";
        if (edge.status === "healthy" && edge.latency < 10) {
          classes += " connection-active";
        }
        if (edge.type === "pod-to-external") {
          classes += " data-flow";
        }
        return classes;
      },
      markers: {
        enabled: true,
        type: "arrow",
      },
      bidirectional: {
        enabled: true,
        mode: "curved",
        curvature: 0.2,
        offset: 5,
      },
      ...config,
    };
  }

  init(context: VisualizationContext): void {
    // Create arrow markers for directed edges
    if (this.config.markers.enabled) {
      this.createMarkers(context);
    }

    // Listen for simulation ticks to update positions
    context.eventBus.on("simulation:tick", () => {
      this.updatePositions();
    });
  }

  private createMarkers(context: VisualizationContext): void {
    const defs = context.svg.select("defs").empty()
      ? context.svg.append("defs")
      : context.svg.select("defs");

    // Create different markers for different edge types
    const markerConfigs = [
      { id: "arrowhead-default", color: "#999", size: 6 },
      { id: "arrowhead-healthy", color: "#10b981", size: 6 },
      { id: "arrowhead-unhealthy", color: "#ef4444", size: 6 },
      { id: "arrowhead-warning", color: "#f59e0b", size: 6 },
    ];

    markerConfigs.forEach(({ id, color, size }) => {
      const marker = defs
        .append("marker")
        .attr("id", id)
        .attr("viewBox", "0 -5 10 10")
        .attr("refX", 12)
        .attr("refY", 0)
        .attr("markerWidth", size)
        .attr("markerHeight", size)
        .attr("orient", "auto");

      marker.append("path").attr("d", "M0,-5L5,0L0,5").attr("fill", color);
    });
  }

  private getMarkerId(edge: NetworkLink): string {
    if (!this.config.markers.enabled) return "";

    if (edge.status === "unhealthy") return `url(#arrowhead-unhealthy)`;
    if (edge.latency < 5) return `url(#arrowhead-healthy)`;
    if (edge.latency < 50) return `url(#arrowhead-warning)`;
    return `url(#arrowhead-default)`;
  }

  update(data: NetworkData, context: VisualizationContext): void {
    const edgesLayer = context.layers.get("edges");
    if (!edgesLayer) {
      console.warn("Edges layer not found");
      return;
    }

    // Clear previous edges
    edgesLayer.selectAll(".network-edge").remove();

    // Separate bidirectional and unidirectional edges
    const bidirectionalEdges = data.links.filter(
      (link) => link.bidirectional && this.config.bidirectional.enabled
    );
    const unidirectionalEdges = data.links.filter(
      (link) => !link.bidirectional || !this.config.bidirectional.enabled
    );

    // Render unidirectional edges as lines
    this.renderStraightEdges(unidirectionalEdges, edgesLayer, context);

    // Render bidirectional edges based on mode
    if (this.config.bidirectional.enabled) {
      this.renderCurvedEdges(bidirectionalEdges, edgesLayer, context);
    }

    // Update edge selection for event handling
    this.edgeSelection = edgesLayer.selectAll(".network-edge");
    this.setupEventHandlers(context);
  }

  private renderStraightEdges(
    edges: NetworkLink[],
    layer: d3.Selection<SVGGElement, unknown, null, undefined>,
    context: VisualizationContext
  ): void {
    const lines = layer
      .selectAll<SVGLineElement, NetworkLink>(".network-edge-line")
      .data(edges, (d) => `${getNodeId(d.source)}-${getNodeId(d.target)}`);

    const enterLines = lines
      .enter()
      .append("line")
      .attr(
        "class",
        (d) => `network-edge network-edge-line ${this.config.className(d)}`
      )
      .style("opacity", 0)
      .style("cursor", "pointer");

    const allLines = enterLines.merge(lines);

    allLines
      .transition()
      .duration(context.config.animation.duration)
      .attr("stroke", this.config.color)
      .attr("stroke-width", this.config.width)
      .style("opacity", this.config.opacity)
      .attr("marker-end", this.getMarkerId.bind(this));
  }

  private renderCurvedEdges(
    edges: NetworkLink[],
    layer: d3.Selection<SVGGElement, unknown, null, undefined>,
    context: VisualizationContext
  ): void {
    const paths = layer
      .selectAll<SVGPathElement, NetworkLink>(".network-edge-path")
      .data(edges, (d) => `${getNodeId(d.source)}-${getNodeId(d.target)}`);

    const enterPaths = paths
      .enter()
      .append("path")
      .attr(
        "class",
        (d) => `network-edge network-edge-path ${this.config.className(d)}`
      )
      .style("opacity", 0)
      .style("cursor", "pointer")
      .attr("fill", "none");

    const allPaths = enterPaths.merge(paths);

    allPaths
      .transition()
      .duration(context.config.animation.duration)
      .attr("stroke", this.config.color)
      .attr("stroke-width", this.config.width)
      .style("opacity", this.config.opacity)
      .attr("d", this.calculateCurvedPath.bind(this));
  }

  private calculateCurvedPath(edge: NetworkLink): string {
    const source = typeof edge.source === "string" ? null : edge.source;
    const target = typeof edge.target === "string" ? null : edge.target;

    if (
      !source ||
      !target ||
      source.x == null ||
      source.y == null ||
      target.x == null ||
      target.y == null
    ) {
      return "";
    }

    const dx = target.x - source.x;
    const dy = target.y - source.y;
    const dr = Math.sqrt(dx * dx + dy * dy);

    if (dr === 0) return "";

    // Calculate control point offset perpendicular to line
    const curvature = this.config.bidirectional.curvature;
    const offset = dr * curvature;

    // Perpendicular vector for offset
    const perpX = (-dy / dr) * offset;
    const perpY = (dx / dr) * offset;

    // Control point
    const controlX = (source.x + target.x) / 2 + perpX;
    const controlY = (source.y + target.y) / 2 + perpY;

    return `M${source.x},${source.y} Q${controlX},${controlY} ${target.x},${target.y}`;
  }

  private calculateParallelOffset(edge: any): {
    x1: number;
    y1: number;
    x2: number;
    y2: number;
  } {
    const source = typeof edge.source === "string" ? null : edge.source;
    const target = typeof edge.target === "string" ? null : edge.target;

    if (
      !source ||
      !target ||
      source.x == null ||
      source.y == null ||
      target.x == null ||
      target.y == null
    ) {
      return { x1: 0, y1: 0, x2: 0, y2: 0 };
    }

    const dx = target.x - source.x;
    const dy = target.y - source.y;
    const length = Math.sqrt(dx * dx + dy * dy);

    if (length === 0)
      return { x1: source.x, y1: source.y, x2: target.x, y2: target.y };

    // Perpendicular offset
    const offset = this.config.bidirectional.offset;
    const offsetDirection = edge.__direction === "forward" ? 1 : -1;
    const perpX = (-dy / length) * offset * offsetDirection;
    const perpY = (dx / length) * offset * offsetDirection;

    return {
      x1: source.x + perpX,
      y1: source.y + perpY,
      x2: target.x + perpX,
      y2: target.y + perpY,
    };
  }

  private setupEventHandlers(context: VisualizationContext): void {
    if (!this.edgeSelection) return;

    this.edgeSelection
      .on("click", (event, edge) => {
        event.stopPropagation();
        const edgeEvent: EdgeEvent = {
          type: "click",
          edge,
          event,
        };
        context.eventBus.emit("edge:click", edgeEvent);
      })
      .on("mouseenter", (event, edge) => {
        // Apply enhanced hover effect for bidirectional edges
        const element = d3.select(event.target);
        const isBidirectional =
          edge.bidirectional && this.config.bidirectional.enabled;

        element
          .transition()
          .duration(150)
          .attr("stroke", isBidirectional ? "#0066cc" : "#000000")
          .attr(
            "stroke-width",
            isBidirectional
              ? this.config.width(edge) * 1.5
              : this.config.width(edge) * 1.2
          )
          .style("opacity", 1);

        // Add visual feedback for bidirectional connections
        if (isBidirectional && edge.reverseLink) {
          // Briefly highlight the reverse connection metrics
          console.log(
            `Bidirectional connection: ${getNodeId(edge.source)} ↔ ${getNodeId(edge.target)}`
          );
          console.log(`Forward: ${edge.latency}ms (${edge.status})`);
          console.log(
            `Reverse: ${edge.reverseLink.latency}ms (${edge.reverseLink.status})`
          );
        }

        const edgeEvent: EdgeEvent = {
          type: "hover",
          edge,
          event,
        };
        context.eventBus.emit("edge:hover", edgeEvent);
      })
      .on("mouseleave", (event, edge) => {
        // Remove hover effect and restore original styling
        d3.select(event.target)
          .transition()
          .duration(150)
          .attr("stroke", this.config.color(edge))
          .attr("stroke-width", this.config.width(edge))
          .style("opacity", this.config.opacity(edge));

        context.eventBus.emit("edge:hover:end", { edge, event });
      });
  }

  private updatePositions(): void {
    if (!this.edgeSelection) return;

    // Update straight line positions
    this.edgeSelection
      .filter(".network-edge-line")
      .attr("x1", (d: any) => {
        const source = typeof d.source === "string" ? null : d.source;
        return source?.x || 0;
      })
      .attr("y1", (d: any) => {
        const source = typeof d.source === "string" ? null : d.source;
        return source?.y || 0;
      })
      .attr("x2", (d: any) => {
        const target = typeof d.target === "string" ? null : d.target;
        return target?.x || 0;
      })
      .attr("y2", (d: any) => {
        const target = typeof d.target === "string" ? null : d.target;
        return target?.y || 0;
      });

    // Update curved path positions
    this.edgeSelection
      .filter(".network-edge-path")
      .attr("d", this.calculateCurvedPath.bind(this));

    // Update parallel line positions
    this.edgeSelection
      .filter(".network-edge-parallel")
      .each((d: any, i, nodes) => {
        const offset = this.calculateParallelOffset(d);
        d3.select(nodes[i])
          .attr("x1", offset.x1)
          .attr("y1", offset.y1)
          .attr("x2", offset.x2)
          .attr("y2", offset.y2);
      });

    // Update bundled line positions (same as straight lines)
    this.edgeSelection
      .filter(".network-edge-bundled")
      .attr("x1", (d: any) => {
        const source = typeof d.source === "string" ? null : d.source;
        return source?.x || 0;
      })
      .attr("y1", (d: any) => {
        const source = typeof d.source === "string" ? null : d.source;
        return source?.y || 0;
      })
      .attr("x2", (d: any) => {
        const target = typeof d.target === "string" ? null : d.target;
        return target?.x || 0;
      })
      .attr("y2", (d: any) => {
        const target = typeof d.target === "string" ? null : d.target;
        return target?.y || 0;
      });
  }

  destroy(context: VisualizationContext): void {
    const edgesLayer = context.layers.get("edges");
    if (edgesLayer) {
      edgesLayer.selectAll(".network-edge").remove();
    }
    this.edgeSelection = undefined;
  }

  getConfig(): EdgePluginConfig {
    return this.config;
  }

  updateConfig(newConfig: Partial<EdgePluginConfig>): void {
    this.config = { ...this.config, ...newConfig };

    // Re-apply styles if edges exist
    if (this.edgeSelection) {
      this.edgeSelection
        .attr("stroke", this.config.color)
        .attr("stroke-width", this.config.width)
        .style("opacity", this.config.opacity)
        .attr("class", (d: any) => `network-edge ${this.config.className(d)}`)
        .attr("marker-end", this.getMarkerId.bind(this));
    }
  }

  // Utility methods
  highlightEdge(sourceId: string, targetId: string): void {
    if (!this.edgeSelection) return;

    this.edgeSelection
      .filter((d: any) => {
        const edgeSourceId = getNodeId(d.source);
        const edgeTargetId = getNodeId(d.target);

        // Highlight both directions for bidirectional edges
        return (
          (edgeSourceId === sourceId && edgeTargetId === targetId) ||
          (d.bidirectional &&
            edgeSourceId === targetId &&
            edgeTargetId === sourceId)
        );
      })
      .transition()
      .duration(200)
      .attr("stroke", "#ff6b35")
      .attr("stroke-width", 4)
      .style("opacity", 1);
  }

  unhighlightEdge(sourceId: string, targetId: string): void {
    if (!this.edgeSelection) return;

    this.edgeSelection
      .filter(
        (d: any) =>
          getNodeId(d.source) === sourceId && getNodeId(d.target) === targetId
      )
      .transition()
      .duration(200)
      .attr("stroke", this.config.color)
      .attr("stroke-width", this.config.width)
      .style("opacity", this.config.opacity);
  }

  highlightNeighbors(nodeId: string): void {
    if (!this.edgeSelection) return;

    // Highlight all edges connected to the node
    this.edgeSelection.style("opacity", (d: any) => {
      const sourceId = getNodeId(d.source);
      const targetId = getNodeId(d.target);
      return sourceId === nodeId || targetId === nodeId ? 1 : 0.1;
    });
  }

  unhighlightAll(): void {
    if (!this.edgeSelection) return;

    this.edgeSelection
      .transition()
      .duration(200)
      .style("opacity", this.config.opacity);
  }
}
