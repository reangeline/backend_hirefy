terraform {
  backend "s3" {
    bucket = "applywise-tf-state-034362044245"
    key    = "dev/terraform.tfstate"
    region = "us-east-1"
  }
}
