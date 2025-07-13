import type {
  GoldpingerData,
  NetworkData,
  NetworkNode,
  NetworkLink,
} from "../types/goldpinger";

function processBidirectionalLinks(links: NetworkLink[]): NetworkLink[] {
  const processedLinks: NetworkLink[] = [];
  const linkMap = new Map<string, NetworkLink>();
  const processedPairs = new Set<string>();

  // Group links by source-target pairs
  links.forEach((link) => {
    const sourceId =
      typeof link.source === "string" ? link.source : link.source.id;
    const targetId =
      typeof link.target === "string" ? link.target : link.target.id;
    const linkKey = `${sourceId}-${targetId}`;
    linkMap.set(linkKey, link);
  });

  // Process each link and look for bidirectional pairs
  links.forEach((link) => {
    const sourceId =
      typeof link.source === "string" ? link.source : link.source.id;
    const targetId =
      typeof link.target === "string" ? link.target : link.target.id;
    const forwardKey = `${sourceId}-${targetId}`;
    const reverseKey = `${targetId}-${sourceId}`;

    // Skip if this pair has already been processed
    if (processedPairs.has(forwardKey) || processedPairs.has(reverseKey)) {
      return;
    }

    const reverseLink = linkMap.get(reverseKey);

    if (
      reverseLink &&
      link.type === "pod-to-pod" &&
      reverseLink.type === "pod-to-pod"
    ) {
      // Bidirectional connection found - create primary link with bidirectional data
      const bidirectionalLink: NetworkLink = {
        ...link,
        bidirectional: true,
        direction: "both",
        reverseLink: reverseLink,
      };

      processedLinks.push(bidirectionalLink);
      processedPairs.add(forwardKey);
      processedPairs.add(reverseKey);
    } else {
      // Unidirectional connection
      const unidirectionalLink: NetworkLink = {
        ...link,
        bidirectional: false,
        direction: "forward",
      };

      processedLinks.push(unidirectionalLink);
      processedPairs.add(forwardKey);
    }
  });

  return processedLinks;
}

export function transformGoldpingerData(data: GoldpingerData): NetworkData {
  const nodes: NetworkNode[] = [];
  const links: NetworkLink[] = [];
  const nodeIds = new Set<string>();

  // Create pod nodes from hosts data
  data.hosts.forEach((host) => {
    const podName = host.podName;
    const responseData = data.responses[podName];

    nodes.push({
      id: podName,
      label: `${podName}\n${host.podIP}`,
      type: "pod",
      hostIP: host.hostIP,
      podIP: host.podIP,
      podName: host.podName,
      status: responseData?.OK ? "healthy" : "unhealthy",
      data: host,
    });
    nodeIds.add(podName);
  });

  // Create external target nodes from probeResults
  Object.keys(data.probeResults).forEach((target) => {
    if (!nodeIds.has(target)) {
      const targetData = data.probeResults[target];
      const allResults = Object.values(targetData).flat();
      const hasErrors = allResults.some((result) => "error" in result);

      nodes.push({
        id: target,
        label: target,
        type: "external",
        status: hasErrors ? "unhealthy" : "healthy",
        data: targetData,
      });
      nodeIds.add(target);
    }
  });

  // Create pod-to-pod links from responses
  Object.entries(data.responses).forEach(([sourcePod, responseData]) => {
    if (responseData.response?.podResults) {
      Object.entries(responseData.response.podResults).forEach(
        ([targetPod, result]) => {
          if (sourcePod !== targetPod) {
            links.push({
              source: sourcePod,
              target: targetPod,
              latency: result["response-time-ms"] || 0,
              status: result.OK ? "healthy" : "unhealthy",
              type: "pod-to-pod",
              data: result,
            });
          }
        }
      );
    }
  });

  // Create pod-to-external links from probeResults
  Object.entries(data.probeResults).forEach(([target, podResults]) => {
    Object.entries(podResults).forEach(([podName, results]) => {
      const result = results[0]; // Take first result if multiple
      if (result) {
        links.push({
          source: podName,
          target,
          latency: result["response-time-ms"] || 0,
          status: "error" in result ? "unhealthy" : "healthy",
          type: "pod-to-external",
          data: result,
        });
      }
    });
  });

  // Process bidirectional connections
  const processedLinks = processBidirectionalLinks(links);

  return { nodes, links: processedLinks };
}

export function getNodeColor(node: NetworkNode): string {
  switch (node.status) {
    case "healthy":
      return node.type === "pod" ? "#10b981" : "#3b82f6"; // green for pods, blue for external
    case "unhealthy":
      return "#ef4444"; // red
    default:
      return "#6b7280"; // gray
  }
}

export function getLinkColor(link: NetworkLink): string {
  if (link.status === "unhealthy") {
    return "#ef4444"; // red
  }

  // Color based on latency for healthy connections
  if (link.latency < 5) {
    return "#10b981"; // green - fast
  } else if (link.latency < 50) {
    return "#f59e0b"; // yellow - moderate
  } else {
    return "#f97316"; // orange - slow
  }
}

export function getLinkWidth(link: NetworkLink): number {
  // Base width
  let width = 2;

  // Adjust based on latency (inverse relationship - faster = thicker)
  if (link.latency < 5) {
    width = 3;
  } else if (link.latency > 100) {
    width = 1;
  }

  return width;
}
