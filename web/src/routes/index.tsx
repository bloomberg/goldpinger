import { createFileRoute } from '@tanstack/react-router'
export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h2 className="text-3xl font-bold text-gray-900 mb-6">
          Goldpinger Network Monitoring Dashboard
        </h2>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h3 className="text-xl font-semibold text-gray-900 mb-3">
              Network Topology
            </h3>
            <p className="text-gray-600 mb-4">
              Visualize your Kubernetes cluster connectivity with an interactive D3.js network graph.
              See pod-to-pod connections, external target reachability, and latency information.
            </p>
            <a 
              href="/network"
              className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              View Network Graph →
            </a>
          </div>
          
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h3 className="text-xl font-semibold text-gray-900 mb-3">
              Real-time Monitoring
            </h3>
            <p className="text-gray-600 mb-4">
              Monitor connectivity between pods in real-time. Goldpinger continuously tests
              network paths and provides immediate feedback on cluster health.
            </p>
            <div className="text-sm text-gray-500">
              Feature coming soon...
            </div>
          </div>
        </div>

        <div className="bg-blue-50 p-6 rounded-lg border border-blue-200">
          <h3 className="text-lg font-semibold text-blue-900 mb-2">
            About Goldpinger
          </h3>
          <p className="text-blue-800">
            Goldpinger is a debugging tool for Kubernetes networking. It runs as a DaemonSet
            and pings between all pods to help identify networking issues. This UI provides
            a modern React-based interface for visualizing the connectivity data.
          </p>
        </div>
      </div>
    </div>
  )
}
