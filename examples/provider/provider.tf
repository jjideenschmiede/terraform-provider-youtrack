terraform {
  required_providers {
    youtrack = {
      source = "jjideenschmiede/youtrack"
    }
  }
}

provider "youtrack" {
  url   = "https://example.youtrack.cloud"
  token = var.youtrack_token
}
