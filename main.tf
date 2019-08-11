terraform {
  required_version = "0.12.4"
  required_providers {
    null = "2.1.1"
  }
}

provider "aws" {
  version = "2.11.0"
  region  = "ap-northeast-1"
}

resource "aws_security_group" "hoge" {
  name = "hoge"
  egress {
    from_port = 0
    to_port   = 0
    protocol  = -1
  }
}
