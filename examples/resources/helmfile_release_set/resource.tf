resource "helmfile_release_set" "mystack" {
  # Enable Go template rendering (for helmfile v0.150.0+)
  enable_go_template = true

  content = <<EOF
repositories:
- name: stable
  url: https://charts.helm.sh/stable

releases:
- name: myapp
  chart: stable/nginx-ingress
  version: 1.41.3
  values:
  - controller:
      service:
        type: LoadBalancer
EOF

  # Path to kubeconfig file
  kubeconfig = pathexpand("~/.kube/config")

  # Working directory for helmfile operations
  working_directory = path.module

  # Helmfile environment to deploy
  environment = "default"

  # Environment variables for helmfile
  environment_variables = {
    ENVIRONMENT = "production"
  }

  # State values to pass to helmfile
  values = [
    yamlencode({
      region = "us-west-2"
    })
  ]

  # Label selector to filter releases
  selector = {
    tier = "frontend"
  }

  # Number of concurrent helm processes (0 = unlimited)
  concurrency = 5

  # Specify helmfile and helm versions
  version      = "0.150.0"
  helm_version = "3.12.0"

  # Optional: Enable dry_run to render templates without deploying
  # When enabled, runs 'helmfile template' instead of 'helmfile apply'
  # The rendered manifests will be available in the 'template_output' attribute
  # dry_run = true
}
