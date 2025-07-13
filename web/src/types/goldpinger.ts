export interface Host {
  hostIP: string;
  podIP: string;
  podName: string;
}

export interface ProbeResult {
  protocol: string;
  "response-time-ms": number;
  error?: string;
}

export interface PodResult {
  HostIP: string;
  OK: boolean;
  PingTime: string;
  PodIP: string;
  response: {
    boot_time: string;
  };
  "response-time-ms": number;
  "status-code": number;
}

export interface ResponseData {
  HostIP: string;
  OK: boolean;
  PodIP: string;
  response: {
    podResults: Record<string, PodResult>;
    probeResults: Record<string, ProbeResult[]>;
  };
}

export interface GoldpingerData {
  hosts: Host[];
  probeResults: Record<string, Record<string, ProbeResult[]>>;
  responses: Record<string, ResponseData>;
}

export interface NetworkNode {
  id: string;
  label: string;
  x?: number;
  y?: number;
  fx?: number | null;
  fy?: number | null;
  type: "pod" | "external";
  hostIP?: string;
  podIP?: string;
  podName?: string;
  status: "healthy" | "unhealthy" | "unknown";
  data: Host | ResponseData | Record<string, ProbeResult[]>;
}

export interface NetworkLink {
  source: string | NetworkNode; // D3.js mutates this from string ID to NetworkNode object
  target: string | NetworkNode; // D3.js mutates this from string ID to NetworkNode object
  latency: number;
  status: "healthy" | "unhealthy" | "unknown";
  type: "pod-to-pod" | "pod-to-external";
  data: PodResult | ProbeResult;
  // Bidirectional support
  bidirectional?: boolean;
  reverseLink?: NetworkLink;
  direction?: "forward" | "reverse" | "both";
}

export interface NetworkData {
  nodes: NetworkNode[];
  links: NetworkLink[];
}
