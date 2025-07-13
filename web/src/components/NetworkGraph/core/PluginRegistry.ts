import type { VisualizationPlugin } from '../types';

export interface PluginFactory<T extends VisualizationPlugin = VisualizationPlugin> {
  create(config?: any): T;
  defaultConfig: any;
  description: string;
}

export class PluginRegistry {
  private static instance: PluginRegistry;
  private factories = new Map<string, PluginFactory>();

  static getInstance(): PluginRegistry {
    if (!PluginRegistry.instance) {
      PluginRegistry.instance = new PluginRegistry();
    }
    return PluginRegistry.instance;
  }

  register<T extends VisualizationPlugin>(
    name: string,
    factory: PluginFactory<T>
  ): void {
    if (this.factories.has(name)) {
      console.warn(`Plugin ${name} is already registered. Overwriting.`);
    }
    this.factories.set(name, factory);
  }

  create<T extends VisualizationPlugin>(
    name: string,
    config?: any
  ): T {
    const factory = this.factories.get(name);
    if (!factory) {
      throw new Error(`Plugin ${name} is not registered`);
    }
    return factory.create(config) as T;
  }

  getDefaultConfig(name: string): any {
    const factory = this.factories.get(name);
    if (!factory) {
      throw new Error(`Plugin ${name} is not registered`);
    }
    return factory.defaultConfig;
  }

  getAvailablePlugins(): string[] {
    return Array.from(this.factories.keys());
  }

  getPluginInfo(name: string): { description: string; defaultConfig: any } | null {
    const factory = this.factories.get(name);
    if (!factory) return null;
    
    return {
      description: factory.description,
      defaultConfig: factory.defaultConfig,
    };
  }

  unregister(name: string): boolean {
    return this.factories.delete(name);
  }

  clear(): void {
    this.factories.clear();
  }
}