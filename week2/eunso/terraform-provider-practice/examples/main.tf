terraform {
  required_providers {
    localnote = {
      source = "registry.terraform.io/study/localnote"
    }
  }
}

provider "localnote" {}

resource "localnote_note" "hello" {
  path    = "/tmp/hello.txt"
  content = "안녕하세요, Terraform Provider 실습입니다!"
}

output "note_path" {
  value = localnote_note.hello.path
}
