# -*- mode: bazel-starlark -*-

# Build the image
docker_build('goldpinger-local', '.')

# Deploy with Helm
k8s_yaml(
  helm(
    'charts/goldpinger',
    set = [
      # Set the image to the one built by Tilt
      'image.repository=goldpinger-local',
    ],
  )
)

# Track Goldpinger Resource
k8s_resource(
  'chart-goldpinger',
  port_forwards = [8080],
)

# Validate that all 2 nodes can talk to eachother
local_resource(
  'check_reachability',
  cmd='./extras/check_reachability.sh 2',
  resource_deps = [
    'chart-goldpinger',
  ],
)
