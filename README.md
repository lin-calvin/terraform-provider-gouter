# Terraform Provider for Gouter

Terraform/OpenTofu provider for [Gouter](https://github.com/lin-calvin/gouter) — a user-space SD-WAN router.

## Usage

```hcl
terraform {
  required_providers {
    gouter = {
      source = "github.com/lin-calvin/terraform-provider-gouter"
    }
  }
}

provider "gouter" {
  endpoint = "http://127.0.0.1:8081/api/v1"
}

resource "gouter_route" "example" {
  prefix   = "10.0.0.0/24"
  next_hop = "172.22.138.1"
  export   = true
}

resource "gouter_link" "home" {
  name    = "home"
  address = "172.22.138.40/32"
  peer_ip = "172.22.138.38"
  wg_listen_port = 51829
  wg_private_key = var.wg_private_key
  wg_public_key  = var.wg_public_key
  wg_allowed_ips = "172.16.0.0/12"
}

resource "gouter_bgp_peer" "home" {
  name     = "home"
  address  = "172.22.138.38"
  asn      = 4242423166
  families = ["ipv4-unicast"]
}
```

## Development

See [Gouter](https://github.com/lin-calvin/gouter) for the router implementation.
