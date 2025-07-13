import type { VisualizationConfig } from "../types";

export interface NetworkGraphConfig extends VisualizationConfig {
  plugins: {
    [pluginName: string]: {
      enabled: boolean;
      config: any;
    };
  };
  theme: {
    name: string;
    colors: {
      background: string;
      primary: string;
      secondary: string;
      success: string;
      warning: string;
      error: string;
      text: string;
    };
  };
}

export const DEFAULT_CONFIG: NetworkGraphConfig = {
  width: 800,
  height: 600,
  margins: { top: 20, right: 20, bottom: 20, left: 20 },
  zoom: {
    enabled: true,
    scaleExtent: [0.1, 4],
  },
  animation: {
    duration: 300,
    easing: "ease-in-out",
  },
  plugins: {
    "force-simulation": {
      enabled: true,
      config: {
        enabled: true,
        forces: {
          link: { distance: 200, strength: 0.5 },
          charge: { strength: -200 },
          center: { x: 0, y: 0 },
          collision: { radius: 25 },
        },
      },
    },
    "node-render": {
      enabled: true,
      config: {
        // Default node rendering config will be applied by the plugin
      },
    },
    "edge-render": {
      enabled: true,
      config: {
        markers: { enabled: true, type: "arrow" },
      },
    },
    interaction: {
      enabled: true,
      config: {
        drag: { enabled: true },
        zoom: { enabled: true, scaleExtent: [0.1, 4] },
        hover: { enabled: true, highlightNeighbors: true },
        click: { enabled: true },
      },
    },
    labels: {
      enabled: true,
      config: {
        fontSize: 12,
        fontFamily: "Arial, sans-serif",
        color: "#333333",
      },
    },
  },
  theme: {
    name: "default",
    colors: {
      background: "#ffffff",
      primary: "#3b82f6",
      secondary: "#6b7280",
      success: "#10b981",
      warning: "#f59e0b",
      error: "#ef4444",
      text: "#111827",
    },
  },
};

export class ConfigurationManager {
  private config: NetworkGraphConfig;
  private changeListeners: ((config: NetworkGraphConfig) => void)[] = [];

  constructor(initialConfig?: Partial<NetworkGraphConfig>) {
    console.log(
      "Initializing ConfigurationManager with config:",
      initialConfig
    );
    this.config = this.mergeConfig(DEFAULT_CONFIG, initialConfig || {});
  }

  private mergeConfig(
    base: NetworkGraphConfig,
    override: Partial<NetworkGraphConfig>
  ): NetworkGraphConfig {
    const merged = { ...base };

    // Merge top-level properties
    Object.keys(override).forEach((key) => {
      if (key === "plugins") {
        // Special handling for plugins
        merged.plugins = { ...base.plugins };
        Object.keys(override.plugins || {}).forEach((pluginName) => {
          merged.plugins[pluginName] = {
            ...base.plugins[pluginName],
            ...override.plugins![pluginName],
            config: {
              ...base.plugins[pluginName]?.config,
              ...override.plugins![pluginName]?.config,
            },
          };
        });
      } else if (key === "theme") {
        // Special handling for theme
        merged.theme = {
          ...base.theme,
          ...override.theme,
          colors: {
            ...base.theme.colors,
            ...override.theme?.colors,
          },
        };
      } else if (
        typeof override[key as keyof NetworkGraphConfig] === "object"
      ) {
        // Deep merge for other objects
        const baseValue = base[key as keyof NetworkGraphConfig];
        const overrideValue = override[key as keyof NetworkGraphConfig];
        
        merged[key as keyof NetworkGraphConfig] = {
          ...(typeof baseValue === "object" && baseValue !== null ? baseValue : {}),
          ...(typeof overrideValue === "object" && overrideValue !== null ? overrideValue : {}),
        } as any;
      } else {
        // Direct assignment for primitives
        (merged as any)[key] = (override as any)[key];
      }
    });

    return merged;
  }

  getConfig(): NetworkGraphConfig {
    return { ...this.config };
  }

  updateConfig(updates: Partial<NetworkGraphConfig>): void {
    this.config = this.mergeConfig(this.config, updates);

    // Notify listeners
    this.changeListeners.forEach((listener) => {
      try {
        listener(this.config);
      } catch (error) {
        console.error("Error in config change listener:", error);
      }
    });
  }

  getPluginConfig(pluginName: string): any {
    return this.config.plugins[pluginName]?.config || {};
  }

  updatePluginConfig(pluginName: string, config: any): void {
    this.updateConfig({
      plugins: {
        [pluginName]: {
          enabled: this.config.plugins[pluginName]?.enabled ?? true,
          config,
        },
      },
    });
  }

  enablePlugin(pluginName: string): void {
    this.updateConfig({
      plugins: {
        [pluginName]: {
          enabled: true,
          config: this.config.plugins[pluginName]?.config || {},
        },
      },
    });
  }

  disablePlugin(pluginName: string): void {
    this.updateConfig({
      plugins: {
        [pluginName]: {
          enabled: false,
          config: this.config.plugins[pluginName]?.config || {},
        },
      },
    });
  }

  isPluginEnabled(pluginName: string): boolean {
    return this.config.plugins[pluginName]?.enabled ?? false;
  }

  setTheme(
    themeName: string,
    colors?: Partial<NetworkGraphConfig["theme"]["colors"]>
  ): void {
    this.updateConfig({
      theme: {
        name: themeName,
        colors: colors
          ? { ...this.config.theme.colors, ...colors }
          : this.config.theme.colors,
      },
    });
  }

  getTheme(): NetworkGraphConfig["theme"] {
    return { ...this.config.theme };
  }

  onConfigChange(listener: (config: NetworkGraphConfig) => void): () => void {
    this.changeListeners.push(listener);

    // Return unsubscribe function
    return () => {
      const index = this.changeListeners.indexOf(listener);
      if (index > -1) {
        this.changeListeners.splice(index, 1);
      }
    };
  }

  reset(): void {
    this.config = { ...DEFAULT_CONFIG };
    this.changeListeners.forEach((listener) => listener(this.config));
  }

  // Utility methods for common configuration patterns
  setDimensions(width: number, height: number): void {
    this.updateConfig({ width, height });
  }

  setZoomEnabled(enabled: boolean): void {
    this.updateConfig({
      zoom: { ...this.config.zoom, enabled },
    });
  }

  setAnimationDuration(duration: number): void {
    this.updateConfig({
      animation: { ...this.config.animation, duration },
    });
  }

  // Export/import configuration
  exportConfig(): string {
    return JSON.stringify(this.config, null, 2);
  }

  importConfig(configJson: string): void {
    try {
      const imported = JSON.parse(configJson);
      this.updateConfig(imported);
    } catch (error) {
      throw new Error("Invalid configuration JSON");
    }
  }
}
