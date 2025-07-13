import * as d3 from 'd3';
import type { NetworkData, NetworkNode } from '../../../types/goldpinger';
import type {
  VisualizationPlugin,
  VisualizationContext,
  InteractionPluginConfig,
  NodeEvent,
  EdgeEvent,
  ViewportEvent
} from '../types';

export class InteractionPlugin implements VisualizationPlugin<InteractionPluginConfig> {
  name = 'interaction';
  dependencies = ['force-simulation']; // Needs simulation for drag behavior
  
  private config: InteractionPluginConfig;
  private dragBehavior?: d3.DragBehavior<any, NetworkNode, any>;
  private zoomBehavior?: d3.ZoomBehavior<SVGSVGElement, unknown>;

  constructor(config?: Partial<InteractionPluginConfig>) {
    this.config = {
      drag: { enabled: true },
      zoom: { enabled: true, scaleExtent: [0.1, 4] },
      hover: { enabled: true, highlightNeighbors: true },
      click: { enabled: true },
      ...config,
    };
  }

  init(context: VisualizationContext): void {
    if (this.config.zoom.enabled) {
      this.setupZoom(context);
    }

    if (this.config.drag.enabled) {
      this.setupDrag(context);
    }

    // Listen for node and edge events to handle interactions
    this.setupEventListeners(context);
  }

  private setupZoom(context: VisualizationContext): void {
    this.zoomBehavior = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent(this.config.zoom.scaleExtent)
      .on('zoom', (event) => {
        context.container.attr('transform', event.transform);
        
        const viewportEvent: ViewportEvent = {
          type: 'zoom',
          transform: event.transform,
          event: event.sourceEvent,
        };
        context.eventBus.emit('viewport:zoom', viewportEvent);
      });

    context.svg.call(this.zoomBehavior);
  }

  private setupDrag(context: VisualizationContext): void {
    this.dragBehavior = d3.drag<any, NetworkNode>()
      .on('start', (event, node) => {
        if (!event.active && context.simulation) {
          context.simulation.alphaTarget(0.3).restart();
        }
        node.fx = node.x;
        node.fy = node.y;

        const nodeEvent: NodeEvent = {
          type: 'dragStart',
          node,
          event: event.sourceEvent,
        };
        context.eventBus.emit('node:dragStart', nodeEvent);
      })
      .on('drag', (event, node) => {
        node.fx = event.x;
        node.fy = event.y;

        const nodeEvent: NodeEvent = {
          type: 'drag',
          node,
          event: event.sourceEvent,
        };
        context.eventBus.emit('node:drag', nodeEvent);
      })
      .on('end', (event, node) => {
        if (!event.active && context.simulation) {
          context.simulation.alphaTarget(0);
        }
        node.fx = null;
        node.fy = null;

        const nodeEvent: NodeEvent = {
          type: 'dragEnd',
          node,
          event: event.sourceEvent,
        };
        context.eventBus.emit('node:dragEnd', nodeEvent);
      });
  }

  private setupEventListeners(context: VisualizationContext): void {
    // Handle node interactions
    if (this.config.hover.enabled) {
      context.eventBus.on('node:hover', (event: NodeEvent) => {
        this.handleNodeHover(event, context);
      });

      context.eventBus.on('node:hover:end', () => {
        this.handleNodeHoverEnd(context);
      });
    }

    if (this.config.click.enabled) {
      context.eventBus.on('node:click', (event: NodeEvent) => {
        // Emit external event for consumer components
        context.eventBus.emit('external:node:click', event.node);
      });

      context.eventBus.on('edge:click', (event: EdgeEvent) => {
        // Emit external event for consumer components
        context.eventBus.emit('external:edge:click', event.edge);
      });
    }

    // Handle viewport clicks (deselection)
    context.svg.on('click', (event) => {
      // Only trigger if clicking on the SVG background
      if (event.target === context.svg.node()) {
        context.eventBus.emit('external:background:click', event);
      }
    });
  }

  private handleNodeHover(event: NodeEvent, context: VisualizationContext): void {
    if (!this.config.hover.highlightNeighbors) return;

    const nodeId = event.node.id;
    
    // Get edge render plugin to highlight neighbors
    const edgePlugin = this.getEdgeRenderPlugin(context);
    if (edgePlugin) {
      edgePlugin.highlightNeighbors(nodeId);
    }

    // Emit hover event for external handlers
    context.eventBus.emit('external:node:hover', event.node);
  }

  private handleNodeHoverEnd(context: VisualizationContext): void {
    if (!this.config.hover.highlightNeighbors) return;

    // Get edge render plugin to unhighlight
    const edgePlugin = this.getEdgeRenderPlugin(context);
    if (edgePlugin) {
      edgePlugin.unhighlightAll();
    }

    context.eventBus.emit('external:node:hover:end');
  }

  private getEdgeRenderPlugin(context: VisualizationContext): any {
    // This is a bit of a hack - in a real system we'd have a plugin registry
    // For now, we'll emit an event that the edge plugin can listen to
    return null;
  }

  update(data: NetworkData, context: VisualizationContext): void {
    // Apply drag behavior to nodes after they're created
    if (this.config.drag.enabled && this.dragBehavior) {
      // We need to wait for the node render plugin to create the nodes
      setTimeout(() => {
        const nodesLayer = context.layers.get('nodes');
        if (nodesLayer) {
          nodesLayer.selectAll('circle').call(this.dragBehavior! as any);
        }
      }, 0);
    }
  }

  destroy(context: VisualizationContext): void {
    // Remove zoom behavior
    if (this.zoomBehavior) {
      context.svg.on('.zoom', null);
      this.zoomBehavior = undefined;
    }

    // Remove drag behavior
    if (this.dragBehavior) {
      const nodesLayer = context.layers.get('nodes');
      if (nodesLayer) {
        nodesLayer.selectAll('circle').on('.drag', null);
      }
      this.dragBehavior = undefined;
    }

    // Remove click handler
    context.svg.on('click', null);
  }

  getConfig(): InteractionPluginConfig {
    return this.config;
  }

  updateConfig(newConfig: Partial<InteractionPluginConfig>): void {
    this.config = { ...this.config, ...newConfig };
    
    // Update zoom behavior if needed
    if (this.zoomBehavior && newConfig.zoom) {
      this.zoomBehavior.scaleExtent(this.config.zoom.scaleExtent);
    }
  }

  // Utility methods for programmatic control
  zoomTo(scale: number, duration = 300): void {
    if (!this.zoomBehavior) return;
    
    // This would need access to the SVG element to work properly
    // In a real implementation, we'd store the context reference
  }

  resetZoom(duration = 300): void {
    if (!this.zoomBehavior) return;
    
    // Reset to identity transform
    // Implementation would depend on having SVG reference
  }

  enableDrag(): void {
    this.config.drag.enabled = true;
  }

  disableDrag(): void {
    this.config.drag.enabled = false;
  }

  enableZoom(): void {
    this.config.zoom.enabled = true;
  }

  disableZoom(): void {
    this.config.zoom.enabled = false;
  }
}