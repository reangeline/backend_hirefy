terraform {
  backend "s3" {
    bucket = "applywise-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}