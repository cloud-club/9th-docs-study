# Terraform Provider는 어떻게 만들어지는가?

모두가 provider를 `required_providers`에 선언하고 `terraform init`으로 설치해서 쓴다는 건 안다.  
그런데 provider 자체가 어떻게 만들어지고, Terraform core와 어떻게 통신하는지는 잘 모르는 경우가 많다.

---

## Provider는 별도의 바이너리다

Provider는 Terraform core에 내장된 게 아니라 독립적인 Go 바이너리다.  
`terraform init`을 실행하면 Registry에서 해당 바이너리를 다운로드해서 로컬에 저장한다.

```
~/.terraform.d/plugins/
└── registry.terraform.io/hashicorp/aws/5.0.0/darwin_arm64/
    └── terraform-provider-aws_v5.0.0
```

`terraform apply`를 실행하면 Terraform core가 이 바이너리를 subprocess로 실행하고, 그 위에서 통신한다.

---

## 통신 방식: gRPC + go-plugin

Terraform core와 provider는 **gRPC**로 통신한다.  
이 구조를 가능하게 하는 것이 HashiCorp의 오픈소스 라이브러리 [go-plugin](https://github.com/hashicorp/go-plugin)이다.

동작 흐름은 다음과 같다:

```
terraform apply
    │
    ├─ provider 바이너리를 subprocess로 실행
    │
    ├─ provider가 로컬 gRPC 서버를 열고 포트를 stdout으로 출력
    │      예) "1|6|tcp|127.0.0.1:12345|grpc"
    │
    └─ Terraform core가 해당 주소로 gRPC 연결
```

이후 Terraform이 `plan`, `apply` 등을 실행할 때마다 gRPC 메시지를 주고받으며 provider가 실제 API를 호출한다.

---

## Provider가 구현해야 하는 것

Provider 바이너리는 Terraform이 정의한 **Plugin Protocol**을 구현해야 한다.  
현재 기준으로 Protocol v5(SDK)와 v6(Framework) 두 가지가 있다.

Provider가 구현하는 주요 RPC:

| RPC | 설명 |
|-----|------|
| `ConfigureProvider` | `provider {}` 블록의 설정값을 받아 API 클라이언트 초기화 |
| `GetProviderSchema` | 리소스/데이터소스 스키마 반환 |
| `ReadResource` | 실제 인프라 상태를 읽어 state와 비교 |
| `PlanResourceChange` | 변경 계획 계산 |
| `ApplyResourceChange` | 실제 생성/수정/삭제 API 호출 |

---

## Provider를 만드는 두 가지 방법

1. Terraform Plugin SDK (구버전)
2. Terraform Plugin Framework (신버전, 권장)

Framework는 타입 안전성과 테스트 편의성이 더 좋아서 신규 provider는 Framework로 작성한다.

---

## Provider의 디렉토리 구조 (예시)

```
terraform-provider-example/
├── main.go                  # plugin.Serve 진입점
├── internal/provider/
│   ├── provider.go          # provider 스키마 및 Configure 구현
│   ├── resource_server.go   # 리소스 CRUD 구현
│   └── data_source_image.go # 데이터소스 Read 구현
```

`provider.go`에서 어떤 리소스와 데이터소스를 제공할지 선언하고,  
각 파일에서 Create/Read/Update/Delete 함수를 구현하면 된다.

---

## 정리

- Provider = **gRPC 서버를 구현한 Go 바이너리**
- Terraform core는 provider를 subprocess로 띄우고 **gRPC로 통신**
- 이 구조 덕분에 provider를 Terraform과 **독립적으로 배포/버전 관리** 가능
- 누구나 Plugin Framework로 custom provider를 만들어 Registry에 배포할 수 있다

---

## 실습: localnote provider 직접 만들기

`terraform-provider-practice/` 폴더에 실제 동작하는 custom provider가 있다.  
로컬 텍스트 파일을 Terraform 리소스로 관리하는 `localnote_note` 리소스를 구현한 예제다.

```
terraform-provider-practice/
├── go.mod
├── main.go                          # providerserver.Serve 진입점
├── internal/provider/
│   ├── provider.go                  # provider 등록 및 리소스 목록 선언
│   └── resource_note.go             # localnote_note 리소스 CRUD 구현
└── examples/
    └── main.tf                      # 실제 사용 예시
```

### 실행 방법

#### 1. 빌드

```bash
cd terraform-provider-practice
go mod tidy
go build -o terraform-provider-localnote .
```

#### 2. 로컬 provider 경로 설정

`~/.terraformrc`에 아래 내용을 추가해 Registry 대신 로컬 바이너리를 사용하도록 한다.

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/study/localnote" = "/path/to/terraform-provider-practice"
  }
  direct {}
}
```

#### 3. 실행

```bash
cd examples
terraform plan
terraform apply
cat /tmp/hello.txt   # "안녕하세요, Terraform Provider 실습입니다!" 출력
terraform destroy
```

### 핵심 포인트

`resource_note.go`의 CRUD 함수가 각각 어떤 gRPC RPC에 대응하는지 확인하면서 읽으면 좋다.

| 함수 | 대응 RPC |
|------|----------|
| `Create` | `ApplyResourceChange` (생성) |
| `Read` | `ReadResource` |
| `Update` | `ApplyResourceChange` (수정) |
| `Delete` | `ApplyResourceChange` (삭제) |
