import * as d3 from "d3";
import type {
  NetworkData,
  NetworkNode,
  NetworkLink,
} from "../../../types/goldpinger";
import type {
  VisualizationPlugin,
  VisualizationContext,
  ForcePluginConfig,
} from "../types";

export class ForceSimulationPlugin
  implements VisualizationPlugin<ForcePluginConfig>
{
  name = "force-simulation";
  private simulation?: d3.Simulation<NetworkNode, NetworkLink>;
  private config: ForcePluginConfig;

  constructor(config?: Partial<ForcePluginConfig>) {
    this.config = {
      enabled: true,
      forces: {
        link: { distance: 200, strength: 0.5 },
        charge: { strength: -200 },
        center: { x: 0, y: 0 }, // Will be set based on viewport
        collision: { radius: 25 },
      },
      ...config,
    };
  }

  init(context: VisualizationContext): void {
    if (!this.config.enabled) return;

    // Set center based on viewport
    this.config.forces.center.x = context.config.width / 2;
    this.config.forces.center.y = context.config.height / 2;

    this.createSimulation(context);
  }

  private createSimulation(context: VisualizationContext): void {
    this.simulation = d3.forceSimulation<NetworkNode>(context.data.nodes);

    // Add forces
    this.simulation
      .force(
        "link",
        d3
          .forceLink<NetworkNode, NetworkLink>()
          .id((d) => d.id)
          .distance(this.config.forces.link.distance)
          .strength(this.config.forces.link.strength)
      )
      .force(
        "charge",
        d3.forceManyBody().strength(this.config.forces.charge.strength)
      )
      .force(
        "center",
        d3.forceCenter(this.config.forces.center.x, this.config.forces.center.y)
      )
      .force(
        "collision",
        d3.forceCollide().radius(this.config.forces.collision.radius)
      );

    // Store simulation in context for other plugins to use
    context.simulation = this.simulation;

    // Set up tick handler
    this.simulation.on("tick", () => {
      context.eventBus.emit("simulation:tick", {
        nodes: context.data.nodes,
        links: context.data.links,
      });
    });

    // Set up end handler
    this.simulation.on("end", () => {
      context.eventBus.emit("simulation:end");
    });
  }

  update(data: NetworkData, context: VisualizationContext): void {
    if (!this.config.enabled || !this.simulation) return;

    // Update nodes
    this.simulation.nodes(data.nodes);

    // Update links
    const linkForce = this.simulation.force("link") as d3.ForceLink<
      NetworkNode,
      NetworkLink
    >;
    if (linkForce) {
      linkForce.links(data.links);
    }

    // Restart simulation with some alpha
    this.simulation.alpha(0.3).restart();
  }

  destroy(context: VisualizationContext): void {
    if (this.simulation) {
      this.simulation.stop();
      this.simulation = undefined;
    }
    context.simulation = undefined;
  }

  getConfig(): ForcePluginConfig {
    return this.config;
  }

  updateConfig(newConfig: Partial<ForcePluginConfig>): void {
    this.config = { ...this.config, ...newConfig };

    if (this.simulation) {
      // Update forces with new config
      const { forces } = this.config;

      const linkForce = this.simulation.force("link") as d3.ForceLink<
        NetworkNode,
        NetworkLink
      >;
      if (linkForce) {
        linkForce.distance(forces.link.distance).strength(forces.link.strength);
      }

      const chargeForce = this.simulation.force(
        "charge"
      ) as d3.ForceManyBody<NetworkNode>;
      if (chargeForce) {
        chargeForce.strength(forces.charge.strength);
      }

      const centerForce = this.simulation.force(
        "center"
      ) as d3.ForceCenter<NetworkNode>;
      if (centerForce) {
        centerForce.x(forces.center.x).y(forces.center.y);
      }

      const collisionForce = this.simulation.force(
        "collision"
      ) as d3.ForceCollide<NetworkNode>;
      if (collisionForce) {
        collisionForce.radius(forces.collision.radius);
      }
    }
  }

  // Utility methods for external control
  start(): void {
    this.simulation?.restart();
  }

  stop(): void {
    this.simulation?.stop();
  }

  setAlpha(alpha: number): void {
    this.simulation?.alpha(alpha);
  }

  reheat(): void {
    this.simulation?.alpha(0.3).restart();
  }
}
