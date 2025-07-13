# NetworkGraph V2 - Modular D3.js Visualization Framework

## Overview

The NetworkGraph component has been completely refactored from a monolithic 200+ line component into a modular, plugin-based architecture. This new design dramatically improves maintainability, testability, and extensibility.

## Architecture Benefits

### Before (Monolithic)
- Single 200+ line React component
- All D3.js logic mixed together
- Hard to test individual features  
- Difficult to add new functionality
- No separation of concerns
- Styling and behavior tightly coupled

### After (Modular)
- Plugin-based architecture
- Each feature is a separate plugin
- Easy to test individual plugins
- Simple to add new features
- Clear separation of concerns
- Configuration-driven approach

## Core Components

### 1. VisualizationFramework
The central orchestrator that manages plugins, layers, and the event system.

```typescript
const framework = new VisualizationFramework(svgElement, config);
framework
  .addPlugin(forceSimulationPlugin)
  .addPlugin(nodeRenderPlugin)
  .addPlugin(edgeRenderPlugin)
  .updateData(networkData);
```

### 2. Plugin System
Each feature is implemented as a plugin with a standard interface:

```typescript
interface VisualizationPlugin {
  name: string;
  dependencies?: string[];
  init(context: VisualizationContext): void;
  update(data: NetworkData, context: VisualizationContext): void;
  destroy(context: VisualizationContext): void;
}
```

### 3. Event Bus
Decoupled communication between plugins using an event system:

```typescript
context.eventBus.on('node:click', (event) => {
  // Handle node click
});

context.eventBus.emit('node:highlight', nodeId);
```

### 4. Configuration Management
Centralized, type-safe configuration with live updates:

```typescript
const configManager = new ConfigurationManager(initialConfig);
configManager.updatePluginConfig('force-simulation', {
  forces: { charge: { strength: -300 } }
});
```

## Available Plugins

### ForceSimulationPlugin
Handles force-directed layout simulation with configurable forces:
- Link force (distance, strength)
- Charge force (repulsion)
- Center force (gravity to center)
- Collision force (prevents overlap)

### NodeRenderPlugin  
Renders network nodes as SVG circles with customizable:
- Size, color, and stroke based on node properties
- Hover effects and animations
- Click and interaction handling
- Health status visualization

### EdgeRenderPlugin
Renders network edges as SVG lines with:
- Color coding based on latency and health
- Animated markers and arrows
- Pulse animations for active connections
- Data flow animations for external connections

### InteractionPlugin
Handles all user interactions:
- Drag and drop node positioning
- Zoom and pan viewport controls
- Click and hover event handling
- Neighbor highlighting

## Adding New Features

### Creating a New Plugin

1. **Implement the plugin interface:**

```typescript
export class MyCustomPlugin implements VisualizationPlugin {
  name = 'my-custom-plugin';
  dependencies = ['node-render']; // Optional dependencies
  
  init(context: VisualizationContext): void {
    // Initialize your plugin
  }
  
  update(data: NetworkData, context: VisualizationContext): void {
    // Update visualization when data changes
  }
  
  destroy(context: VisualizationContext): void {
    // Cleanup resources
  }
}
```

2. **Register the plugin:**

```typescript
const registry = PluginRegistry.getInstance();
registry.register('my-custom-plugin', {
  create: (config) => new MyCustomPlugin(config),
  defaultConfig: { /* default configuration */ },
  description: 'My custom visualization feature'
});
```

3. **Use the plugin:**

```typescript
framework.addPlugin(registry.create('my-custom-plugin', customConfig));
```

### Example: Adding a Minimap Plugin

```typescript
export class MinimapPlugin implements VisualizationPlugin {
  name = 'minimap';
  dependencies = ['node-render', 'edge-render'];
  
  init(context: VisualizationContext): void {
    const minimapLayer = context.getOrCreateLayer('minimap', 10);
    // Create minimap SVG elements
  }
  
  update(data: NetworkData, context: VisualizationContext): void {
    // Update minimap when main graph changes
  }
  
  destroy(context: VisualizationContext): void {
    // Remove minimap elements
  }
}
```

## Configuration System

### Plugin Configuration
Each plugin can have its own configuration that's managed centrally:

```typescript
const config = {
  plugins: {
    'force-simulation': {
      enabled: true,
      config: {
        forces: {
          link: { distance: 100, strength: 0.8 },
          charge: { strength: -250 }
        }
      }
    },
    'node-render': {
      enabled: true,
      config: {
        radius: (node) => node.type === 'pod' ? 15 : 10,
        color: (node) => getCustomColor(node)
      }
    }
  }
};
```

### Runtime Configuration Updates
Configuration can be updated at runtime:

```typescript
// Update specific plugin config
configManager.updatePluginConfig('force-simulation', {
  forces: { charge: { strength: -400 } }
});

// Enable/disable plugins
configManager.disablePlugin('minimap');
configManager.enablePlugin('labels');
```

## Event System

### Built-in Events
The framework emits various events that plugins can listen to:

- `simulation:tick` - Force simulation step
- `node:click`, `node:hover` - Node interactions  
- `edge:click`, `edge:hover` - Edge interactions
- `viewport:zoom`, `viewport:pan` - View changes

### Custom Events
Plugins can emit custom events for inter-plugin communication:

```typescript
// Plugin A emits an event
context.eventBus.emit('custom:data-loaded', processedData);

// Plugin B listens for the event
context.eventBus.on('custom:data-loaded', (data) => {
  // React to the custom event
});
```

## Testing

The modular architecture makes testing much easier:

### Unit Testing Plugins
```typescript
describe('NodeRenderPlugin', () => {
  it('should render nodes correctly', () => {
    const plugin = new NodeRenderPlugin();
    const mockContext = createMockContext();
    
    plugin.init(mockContext);
    plugin.update(testData, mockContext);
    
    expect(mockContext.layers.get('nodes').selectAll('circle')).toHaveLength(5);
  });
});
```

### Integration Testing
```typescript
describe('VisualizationFramework', () => {
  it('should coordinate plugins correctly', () => {
    const framework = new VisualizationFramework(mockSvg, config);
    framework.addPlugin(new NodeRenderPlugin());
    framework.addPlugin(new EdgeRenderPlugin());
    
    framework.updateData(testData);
    
    // Verify plugins work together
  });
});
```

## Performance

### Optimization Strategies
1. **Lazy Plugin Loading**: Only load plugins that are enabled
2. **Event Debouncing**: Batch rapid events like simulation ticks
3. **Selective Updates**: Plugins only update what changed
4. **Memory Management**: Proper cleanup in destroy methods

### Monitoring
Use the debug mode to monitor performance:

```typescript
// Development mode shows debug information
if (process.env.NODE_ENV === 'development') {
  // Debug panel shows active plugins, event counts, etc.
}
```

## Migration Guide

### From Old NetworkGraph
The new NetworkGraphV2 is API-compatible with the old component:

```typescript
// Old way (still works)
<NetworkGraph 
  data={data} 
  onNodeClick={handleNodeClick} 
  onLinkClick={handleLinkClick} 
/>

// New way (with configuration)
<NetworkGraphV2 
  data={data} 
  onNodeClick={handleNodeClick} 
  onLinkClick={handleLinkClick}
  config={{
    plugins: {
      'force-simulation': {
        config: { forces: { charge: { strength: -300 } } }
      }
    }
  }}
/>
```

### Advanced Usage
For full control, use the framework directly:

```typescript
import { createNetworkGraph } from './NetworkGraph';

const { framework, configManager } = createNetworkGraph(svgElement, config);

// Add custom plugins
framework.addPlugin(new MyCustomPlugin());

// Update configuration
configManager.setTheme('dark');

// Update data
framework.updateData(networkData);
```

## Future Enhancements

The modular architecture makes it easy to add:

- **Layout Algorithms**: Circular, hierarchical, force-directed variants
- **Animation System**: Smooth transitions between states  
- **Export Plugins**: SVG, PNG, PDF export capabilities
- **Data Plugins**: Real-time data streaming, filtering
- **Accessibility**: Screen reader support, keyboard navigation
- **Themes**: Dark mode, high contrast, custom color schemes

This architecture provides a solid foundation for building sophisticated network visualizations that can grow and evolve with your needs.