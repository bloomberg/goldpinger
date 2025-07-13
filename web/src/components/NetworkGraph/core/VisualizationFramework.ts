import * as d3 from "d3";
import type { NetworkData } from "../../../types/goldpinger";
import type {
  VisualizationPlugin,
  VisualizationContext,
  VisualizationConfig,
  EventBus,
  LayerConfig,
} from "../types";

export class SimpleEventBus implements EventBus {
  private listeners = new Map<string, Function[]>();

  on<T = any>(event: string, handler: (data: T) => void): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(handler);
  }

  off(event: string, handler?: Function): void {
    if (!this.listeners.has(event)) return;

    if (handler) {
      const handlers = this.listeners.get(event)!;
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    } else {
      this.listeners.delete(event);
    }
  }

  emit<T = any>(event: string, data?: T): void {
    if (!this.listeners.has(event)) return;

    this.listeners.get(event)!.forEach((handler) => {
      try {
        handler(data);
      } catch (error) {
        console.error(`Error in event handler for ${event}:`, error);
      }
    });
  }
}

export class VisualizationFramework {
  private plugins = new Map<string, VisualizationPlugin>();
  private pluginOrder: string[] = [];
  private context: VisualizationContext;
  private defaultLayers: LayerConfig[] = [
    { name: "background", zIndex: 0, enabled: true },
    { name: "edges", zIndex: 1, enabled: true },
    { name: "nodes", zIndex: 2, enabled: true },
    { name: "labels", zIndex: 3, enabled: true },
    { name: "overlay", zIndex: 4, enabled: true },
  ];

  constructor(
    private svgElement: SVGSVGElement,
    private config: VisualizationConfig
  ) {
    this.context = this.createContext();
    this.initializeLayers();
  }

  private createContext(): VisualizationContext {
    const svg = d3.select(this.svgElement);
    const container = svg.append("g").attr("class", "visualization-container");

    return {
      svg,
      container,
      config: this.config,
      data: { nodes: [], links: [] },
      layers: new Map(),
      eventBus: new SimpleEventBus(),
      getOrCreateLayer: this.getOrCreateLayer.bind(this),
    };
  }

  private initializeLayers(): void {
    this.defaultLayers
      .filter((layer) => layer.enabled)
      .sort((a, b) => a.zIndex - b.zIndex)
      .forEach((layer) => {
        const layerGroup = this.context.container
          .append("g")
          .attr("class", `layer-${layer.name}`)
          .style("z-index", layer.zIndex);

        this.context.layers.set(layer.name, layerGroup);
      });
  }

  addPlugin<T extends VisualizationPlugin>(plugin: T): this {
    // Check dependencies
    if (plugin.dependencies) {
      const missingDeps = plugin.dependencies.filter(
        (dep) => !this.plugins.has(dep)
      );
      if (missingDeps.length > 0) {
        throw new Error(
          `Plugin ${plugin.name} has missing dependencies: ${missingDeps.join(", ")}`
        );
      }
    }

    this.plugins.set(plugin.name, plugin);
    this.pluginOrder.push(plugin.name);

    // Initialize the plugin
    plugin.init(this.context);

    return this;
  }

  removePlugin(name: string): this {
    const plugin = this.plugins.get(name);
    if (plugin) {
      plugin.destroy(this.context);
      this.plugins.delete(name);
      this.pluginOrder = this.pluginOrder.filter((p) => p !== name);
    }
    return this;
  }

  getPlugin<T extends VisualizationPlugin>(name: string): T | undefined {
    return this.plugins.get(name) as T;
  }

  updateData(data: NetworkData): this {
    this.context.data = data;

    // Update all plugins in order
    this.pluginOrder.forEach((pluginName) => {
      const plugin = this.plugins.get(pluginName);
      if (plugin) {
        try {
          plugin.update(data, this.context);
        } catch (error) {
          console.error(`Error updating plugin ${pluginName}:`, error);
        }
      }
    });

    return this;
  }

  updateConfig(newConfig: Partial<VisualizationConfig>): this {
    this.context.config = { ...this.context.config, ...newConfig };

    // Update SVG dimensions if changed
    if (newConfig.width || newConfig.height) {
      this.context.svg
        .attr("width", this.context.config.width)
        .attr("height", this.context.config.height);
    }

    return this;
  }

  getContext(): VisualizationContext {
    return this.context;
  }

  destroy(): void {
    // Destroy all plugins
    this.pluginOrder.forEach((pluginName) => {
      const plugin = this.plugins.get(pluginName);
      if (plugin) {
        plugin.destroy(this.context);
      }
    });

    // Clear the SVG
    this.context.svg.selectAll("*").remove();

    // Clear references
    this.plugins.clear();
    this.pluginOrder = [];
  }

  // Utility method to create layer if it doesn't exist
  getOrCreateLayer(
    name: string,
    zIndex = 0
  ): d3.Selection<SVGGElement, unknown, null, undefined> {
    if (!this.context.layers.has(name)) {
      const layer = this.context.container
        .append("g")
        .attr("class", `layer-${name}`)
        .style("z-index", zIndex);

      this.context.layers.set(name, layer);
    }

    return this.context.layers.get(name)!;
  }
}
