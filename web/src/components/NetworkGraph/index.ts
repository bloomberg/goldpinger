import {
  ConfigurationManager,
  NetworkGraphConfig,
} from "./core/ConfigurationManager";
import { PluginRegistry } from "./core/PluginRegistry";
import { VisualizationFramework } from "./core/VisualizationFramework";

// Core framework exports
export { VisualizationFramework } from "./core/VisualizationFramework";
export {
  ConfigurationManager,
  DEFAULT_CONFIG,
} from "./core/ConfigurationManager";
export { PluginRegistry } from "./core/PluginRegistry";

// Plugin exports
export { ForceSimulationPlugin } from "./plugins/ForceSimulationPlugin";
export { NodeRenderPlugin } from "./plugins/NodeRenderPlugin";
export { EdgeRenderPlugin } from "./plugins/EdgeRenderPlugin";
export { InteractionPlugin } from "./plugins/InteractionPlugin";

// Type exports
export type {
  VisualizationConfig,
  VisualizationPlugin,
  VisualizationContext,
  EventBus,
  NodePluginConfig,
  EdgePluginConfig,
  ForcePluginConfig,
  InteractionPluginConfig,
  NodeEvent,
  EdgeEvent,
  ViewportEvent,
} from "./types";

export type { NetworkGraphConfig } from "./core/ConfigurationManager";
export type { PluginFactory } from "./core/PluginRegistry";

// Utility function to create a basic network graph with default plugins
export function createNetworkGraph(
  container: SVGSVGElement,
  config?: Partial<NetworkGraphConfig>
) {
  const configManager = new ConfigurationManager(config);
  const framework = new VisualizationFramework(
    container,
    configManager.getConfig()
  );

  // Add default plugins
  const registry = PluginRegistry.getInstance();

  try {
    framework
      .addPlugin(
        registry.create(
          "force-simulation",
          configManager.getPluginConfig("force-simulation")
        )
      )
      .addPlugin(
        registry.create(
          "edge-render",
          configManager.getPluginConfig("edge-render")
        )
      )
      .addPlugin(
        registry.create(
          "node-render",
          configManager.getPluginConfig("node-render")
        )
      )
      .addPlugin(
        registry.create(
          "interaction",
          configManager.getPluginConfig("interaction")
        )
      )
      .addPlugin(
        registry.create("labels", configManager.getPluginConfig("labels"))
      );
  } catch (error) {
    console.error(
      "Failed to create network graph with default plugins:",
      error
    );
  }

  return {
    framework,
    configManager,
    update: (data: any) => framework.updateData(data),
    destroy: () => framework.destroy(),
  };
}
