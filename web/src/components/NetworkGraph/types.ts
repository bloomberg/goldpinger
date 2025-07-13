import * as d3 from "d3";
import type {
  NetworkData,
  NetworkNode,
  NetworkLink,
} from "../../types/goldpinger";

// Core visualization framework types
export interface VisualizationConfig {
  width: number;
  height: number;
  margins: { top: number; right: number; bottom: number; left: number };
  zoom: {
    enabled: boolean;
    scaleExtent: [number, number];
  };
  animation: {
    duration: number;
    easing: string;
  };
}

export interface LayerConfig {
  name: string;
  zIndex: number;
  enabled: boolean;
}

// Plugin system interfaces
export interface VisualizationPlugin<T = any> {
  name: string;
  dependencies?: string[];
  init(context: VisualizationContext): void;
  update(data: NetworkData, context: VisualizationContext): void;
  destroy(context: VisualizationContext): void;
  getConfig?(): T;
}

export interface VisualizationContext {
  svg: d3.Selection<SVGSVGElement, unknown, null, undefined>;
  container: d3.Selection<SVGGElement, unknown, null, undefined>;
  config: VisualizationConfig;
  data: NetworkData;
  simulation?: d3.Simulation<NetworkNode, NetworkLink>;
  layers: Map<string, d3.Selection<SVGGElement, unknown, null, undefined>>;
  eventBus: EventBus;
  getOrCreateLayer: (
    name: string,
    zIndex?: number
  ) => d3.Selection<SVGGElement, unknown, null, undefined>;
}

// Event system
export interface EventBus {
  on<T = any>(event: string, handler: (data: T) => void): void;
  off(event: string, handler?: Function): void;
  emit<T = any>(event: string, data?: T): void;
}

// Specific plugin configurations
export interface NodePluginConfig {
  radius: (node: NetworkNode) => number;
  color: (node: NetworkNode) => string;
  stroke: (node: NetworkNode) => string;
  strokeWidth: (node: NetworkNode) => number;
  opacity: (node: NetworkNode) => number;
  className: (node: NetworkNode) => string;
}

export interface EdgePluginConfig {
  width: (edge: NetworkLink) => number;
  color: (edge: NetworkLink) => string;
  opacity: (edge: NetworkLink) => number;
  className: (edge: NetworkLink) => string;
  markers: {
    enabled: boolean;
    type: "arrow" | "circle" | "square";
  };
  bidirectional: {
    enabled: boolean;
    mode: "curved" | "parallel" | "bundled";
    curvature: number; // 0-1, amount of curve for arcs
    offset: number; // pixels for parallel lines
  };
}

export interface ForcePluginConfig {
  enabled: boolean;
  forces: {
    link: {
      distance: number;
      strength: number;
    };
    charge: {
      strength: number;
    };
    center: {
      x: number;
      y: number;
    };
    collision: {
      radius: number;
    };
  };
}

export interface InteractionPluginConfig {
  drag: {
    enabled: boolean;
  };
  zoom: {
    enabled: boolean;
    scaleExtent: [number, number];
  };
  hover: {
    enabled: boolean;
    highlightNeighbors: boolean;
  };
  click: {
    enabled: boolean;
  };
}

// Events
export interface NodeEvent {
  type: "click" | "hover" | "dragStart" | "drag" | "dragEnd";
  node: NetworkNode;
  event: Event;
}

export interface EdgeEvent {
  type: "click" | "hover";
  edge: NetworkLink;
  event: Event;
}

export interface ViewportEvent {
  type: "zoom" | "pan";
  transform: d3.ZoomTransform;
  event: Event;
}
