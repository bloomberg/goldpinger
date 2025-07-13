import { useState, useEffect, useCallback } from "react";
import rawData from "../../raw_data.json";
import type { GoldpingerData } from "../types/goldpinger";

interface UseGoldpingerDataOptions {
  autoRefresh?: boolean;
  refreshInterval?: number; // in milliseconds
  endpoint?: string;
}

interface UseGoldpingerDataReturn {
  data: GoldpingerData | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  lastUpdated: Date | null;
}

export function useGoldpingerData(
  options: UseGoldpingerDataOptions = {}
): UseGoldpingerDataReturn {
  const {
    autoRefresh = false,
    refreshInterval = 30000, // 30 seconds default
    endpoint = "/raw_data.json", // fallback to static file for demo
  } = options;

  const [data, setData] = useState<GoldpingerData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setError(null);

      // Try to fetch from the actual Goldpinger API first
      let response: Response;
      try {
        response = await fetch("/check_all", {
          method: "GET",
          headers: {
            Accept: "application/json",
          },
        });
      } catch (apiError) {
        // Fallback to sample data if API is not available
        console.warn(
          "Goldpinger API not available, using sample data:",
          apiError
        );
        response = await fetch(endpoint);
      }

      // TODO: Uncomment this when using real API
      // if (!response.ok) {
      //   throw new Error(`Failed to fetch data: ${response.status} ${response.statusText}`);
      // }

      // TODO: Uncomment this when using real API
      const jsonData: GoldpingerData = rawData as unknown as GoldpingerData; // Use static data for demo
      setData(jsonData);
      setLastUpdated(new Date());
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Unknown error occurred";
      setError(errorMessage);
      console.error("Error fetching Goldpinger data:", err);
    } finally {
      setLoading(false);
    }
  }, [endpoint]);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchData();
  }, [fetchData]);

  // Initial data fetch
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Auto-refresh functionality
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      fetchData();
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, fetchData]);

  return {
    data,
    loading,
    error,
    refresh,
    lastUpdated,
  };
}

// Helper hook for transforming data
export function useNetworkData(options: UseGoldpingerDataOptions = {}) {
  const goldpingerData = useGoldpingerData(options);

  const networkData = goldpingerData.data
    ? transformGoldpingerData(goldpingerData.data)
    : null;

  return {
    ...goldpingerData,
    networkData,
  };
}

// We need to import this here to avoid circular dependency
function transformGoldpingerData(data: GoldpingerData) {
  // This is a simplified version of the transformation
  // In a real app, you'd import this from the utils file
  const nodes = data.hosts.map((host) => ({
    id: host.podName,
    label: `${host.podName}\n${host.podIP}`,
    type: "pod" as const,
    hostIP: host.hostIP,
    podIP: host.podIP,
    podName: host.podName,
    status: data.responses[host.podName]?.OK
      ? ("healthy" as const)
      : ("unhealthy" as const),
    data: host,
  }));

  const links: any[] = [];

  // Add pod-to-pod links
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

  return { nodes, links };
}
