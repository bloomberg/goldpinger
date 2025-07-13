import type { NetworkNode, NetworkLink } from '../types/goldpinger';

/**
 * Safely extracts the node ID from a source/target that could be either a string ID or NetworkNode object.
 * D3.js force simulation mutates link source/target from string IDs to NetworkNode objects.
 */
export function getNodeId(nodeOrId: string | NetworkNode): string {
  return typeof nodeOrId === 'string' ? nodeOrId : nodeOrId.id;
}

/**
 * Safely extracts a display label from a source/target that could be either a string ID or NetworkNode object.
 * D3.js force simulation mutates link source/target from string IDs to NetworkNode objects.
 */
export function getNodeLabel(nodeOrId: string | NetworkNode): string {
  if (typeof nodeOrId === 'string') {
    return nodeOrId; // Just return the ID if it's still a string
  }
  
  // If it's a NetworkNode object, return a meaningful label
  if (nodeOrId.podName) {
    return nodeOrId.podName;
  }
  
  // For external targets, use the first line of the label
  return nodeOrId.label.split('\n')[0];
}

/**
 * Safely extracts the NetworkNode object from a source/target.
 * Returns null if it's still a string ID (before D3 simulation).
 */
export function getNodeObject(nodeOrId: string | NetworkNode): NetworkNode | null {
  return typeof nodeOrId === 'string' ? null : nodeOrId;
}

/**
 * Checks if a link's source/target has been mutated by D3.js force simulation.
 */
export function isLinkMutated(link: NetworkLink): boolean {
  return typeof link.source === 'object' && typeof link.target === 'object';
}

/**
 * Creates a safe display string for a link, handling both string IDs and NetworkNode objects.
 */
export function getLinkDisplayString(link: NetworkLink): string {
  const sourceLabel = getNodeLabel(link.source);
  const targetLabel = getNodeLabel(link.target);
  return `${sourceLabel} → ${targetLabel}`;
}

/**
 * Validates that a link has the expected structure and handles D3.js mutations gracefully.
 */
export function validateLink(link: NetworkLink): {
  isValid: boolean;
  sourceId: string;
  targetId: string;
  sourceLabel: string;
  targetLabel: string;
} {
  try {
    const sourceId = getNodeId(link.source);
    const targetId = getNodeId(link.target);
    const sourceLabel = getNodeLabel(link.source);
    const targetLabel = getNodeLabel(link.target);
    
    return {
      isValid: Boolean(sourceId && targetId),
      sourceId,
      targetId,
      sourceLabel,
      targetLabel,
    };
  } catch (error) {
    console.error('Error validating link:', error);
    return {
      isValid: false,
      sourceId: 'unknown',
      targetId: 'unknown',
      sourceLabel: 'Unknown Source',
      targetLabel: 'Unknown Target',
    };
  }
}