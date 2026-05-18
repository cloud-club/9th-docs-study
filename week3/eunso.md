# Terraform State는 왜 운영의 중심이 되는가?

Terraform을 처음 배울 때 state는 보통 `terraform.tfstate`라는 파일 이름으로 기억된다.
하지만 실제로 state는 단순한 결과물이 아니라, Terraform이 "내가 관리하는 인프라"를 식별하고,
변경 순서를 계산하고, 여러 사람이 같은 대상을 안전하게 다루도록 만드는 운영의 중심 데이터다.

이번 글은 state 문서와 backend 문서를 각각 요약하기보다는, 하나의 질문에서 출발해보려고 한다.

> Terraform은 왜 state를 필요로 하고, 그 state를 어디에 어떻게 두어야 안전한가?

---

## State는 Terraform의 데이터베이스다

Terraform 설정에는 다음과 같은 리소스가 있을 수 있다.

```hcl
resource "aws_instance" "foo" {
  # ...
}
```

하지만 이 설정만으로는 Terraform이 실제 AWS EC2 인스턴스 `i-abcd1234`를 가리킨다는 사실을 알 수 없다.
Terraform은 이 연결 관계를 state에 저장한다.

즉 state의 첫 번째 역할은 **Terraform 주소와 실제 원격 객체 사이의 매핑**이다.

```
aws_instance.foo  <---- state ---->  EC2 instance i-abcd1234
```

이 매핑이 없으면 Terraform은 매번 클라우드 API 전체를 뒤져서 "이 설정이 저 인스턴스인가?"를 추론해야 한다.
태그를 이용해 찾을 수도 있겠지만, 모든 리소스가 태그를 지원하는 것도 아니고 모든 provider가 같은 방식으로 동작하는 것도 아니다.
그래서 Terraform은 provider별 추론 규칙에 기대지 않고, 자기만의 state 구조를 사용한다.

중요한 점은 Terraform이 기본적으로 **하나의 실제 객체는 하나의 리소스 인스턴스에만 연결되어야 한다**고 기대한다는 것이다.
같은 원격 객체를 여러 리소스 주소에 import하면 state의 의미가 모호해지고, Terraform의 동작도 예측하기 어려워진다.

---

## State는 삭제 순서까지 기억한다

State는 단순히 ID만 저장하지 않는다.
Terraform은 리소스 간 의존성 같은 메타데이터도 state에 기록한다.

보통 의존성은 설정 파일에서 계산할 수 있다.
예를 들어 서버가 서브넷에 의존한다면 Terraform은 서브넷보다 서버를 먼저 만들어야 하고,
삭제할 때는 서버를 먼저 지운 뒤 서브넷을 지워야 한다.

그런데 어떤 리소스 블록을 설정 파일에서 삭제했다고 생각해보자.
Terraform은 "설정에는 없지만 state에는 있는 리소스"를 보고 destroy를 계획한다.
문제는 설정 블록이 사라졌기 때문에 의존성 정보를 더 이상 설정에서 읽을 수 없다는 점이다.

이때 state에 남아 있는 마지막 의존성 정보가 필요하다.
Terraform은 state를 통해 삭제 순서를 계산하고, provider alias처럼 리소스가 마지막으로 어떤 provider 설정과 연결되어 있었는지도 추적한다.

State는 그래서 현재 상태의 스냅샷이면서, 동시에 안전하게 변경하기 위한 실행 문맥이다.

---

## State는 성능 캐시이기도 하다

`terraform plan`은 원하는 설정과 실제 인프라의 현재 상태를 비교해야 한다.
기본적으로 Terraform은 plan/apply 시점에 state에 있는 리소스를 provider API로 refresh한다.

작은 인프라에서는 이 방식이 자연스럽다.
하지만 리소스가 많아지면 이야기가 달라진다.
클라우드 API는 리소스를 한 번에 조회하지 못하는 경우가 많고, 호출마다 지연 시간이 있으며, rate limit도 있다.

그래서 state에는 리소스 속성 값의 캐시도 저장된다.
대규모 환경에서 `-refresh=false`나 `-target` 같은 옵션을 쓰는 이유도 결국 "모든 것을 매번 다시 읽는 비용" 때문이다.

다만 여기서 조심해야 할 점이 있다.
State는 편의를 위한 캐시이기도 하지만, 운영에서는 점점 **사실상 기준 데이터**처럼 다뤄진다.
따라서 state가 틀어지면 Terraform이 보는 세계와 실제 세계가 달라진다.

---

## 로컬 state는 시작점일 뿐이다

Terraform의 기본 backend는 `local`이다.
별도 설정을 하지 않으면 state는 현재 작업 디렉터리의 `terraform.tfstate` 파일로 저장된다.

```hcl
terraform {
  backend "local" {
    path = "relative/path/to/terraform.tfstate"
  }
}
```

`local` backend는 로컬 파일시스템에 state를 저장하고, 시스템 API를 이용해 잠금을 수행한다.
혼자 실습하거나 작은 예제를 다룰 때는 충분하다.

하지만 팀으로 운영하는 순간 로컬 state는 위험해진다.

- 사람마다 다른 state를 보고 apply할 수 있다.
- 이전 작업자의 변경을 모른 채 다음 사람이 변경을 적용할 수 있다.
- state 파일에 민감한 값이 들어갈 수 있는데, 이를 로컬 파일이나 Git에 남기기 쉽다.

그래서 팀 환경에서는 remote backend가 사실상 기본 선택이 된다.
Remote backend를 사용하면 여러 사람이 같은 state를 기준으로 작업할 수 있고,
backend가 지원하는 경우 state locking으로 동시에 쓰는 상황을 막을 수 있다.

---

## Backend는 state의 저장소이자 운영 정책이다

Backend는 Terraform state를 어디에 저장할지 정한다.
하지만 단순한 저장 위치 이상의 의미가 있다.

Backend는 다음 질문에 대한 답이다.

- state는 어느 공유 저장소에 둘 것인가?
- 누가 그 state에 접근할 수 있는가?
- 동시에 apply하면 어떻게 막을 것인가?
- state를 잃거나 잘못 덮어썼을 때 복구할 수 있는가?
- 인증 정보는 설정 파일에 남기지 않고 어떻게 주입할 것인가?

Terraform의 backend 설정은 root module의 `terraform` 블록 안에 작성한다.

```hcl
terraform {
  backend "s3" {
    bucket = "mybucket"
    key    = "path/to/my/key"
    region = "us-east-1"
  }
}
```

Backend 설정에는 몇 가지 제약이 있다.

- 하나의 configuration에는 backend block을 하나만 둘 수 있다.
- backend block 안에서는 input variable, local, data source 같은 named value를 참조할 수 없다.
- backend block 안의 값을 다른 설정에서 참조할 수 없다.
- HCP Terraform이나 Terraform Enterprise의 workspace와 `cloud` block으로 연결하는 경우 `backend` block을 함께 쓰지 않는다.

Backend 설정을 변경하면 `terraform init`을 다시 실행해야 한다.
Terraform은 backend 변경을 감지하면 state를 새 backend로 migrate할지 묻는다.
이때도 기존 state 백업을 먼저 떠두는 것이 좋다.

---

## 인증 정보는 backend 설정에 박지 않는다

Remote backend는 보통 접근 자격 증명이 필요하다.
그리고 state 자체에도 민감한 값이 들어갈 수 있다.

문제는 backend 설정이나 `-backend-config`에 민감한 값을 직접 넣으면,
그 값이 `.terraform` 디렉터리나 plan 파일에 평문으로 남을 수 있다는 점이다.
Terraform은 현재 작업 디렉터리의 `.terraform/terraform.tfstate`에 backend 설정을 저장하고,
저장된 plan 파일에도 plan 생성 시점의 backend 설정이 포함된다.

그래서 backend 문서들이 반복해서 권장하는 방식은 같다.

- credential은 환경 변수나 각 플랫폼의 credential file을 사용한다.
- Git에는 backend의 위치 정보 정도만 남기고, 민감한 값은 partial configuration으로 주입한다.
- `.terraform/` 디렉터리는 절대 Git에 올리지 않는다.

예를 들어 backend type만 코드에 남겨두고,
실제 주소나 credential은 `terraform init -backend-config=...` 또는 환경 변수로 제공할 수 있다.

```hcl
terraform {
  backend "consul" {}
}
```

```hcl
# config.consul.tfbackend
address = "demo.consul.io"
path    = "example_app/terraform_state"
scheme  = "https"
```

```bash
terraform init -backend-config="config.consul.tfbackend"
```

---

## Locking은 state를 동시에 쓰지 못하게 하는 장치다

Terraform은 backend가 지원하면 state를 변경할 수 있는 작업에서 자동으로 lock을 잡는다.
Lock을 잡지 못하면 Terraform은 작업을 계속하지 않는다.

이 기능은 단순하지만 중요하다.
두 사람이 동시에 같은 state를 기준으로 apply하면 둘 다 "내가 최신 상태를 보고 있다"고 생각할 수 있다.
그 결과 state가 손상되거나, 한쪽 변경이 다른 쪽 변경을 덮어쓸 수 있다.

`-lock=false`로 잠금을 끌 수는 있지만 권장되지 않는다.
잠금 해제가 실패한 경우에는 `terraform force-unlock`을 사용할 수 있는데,
이 명령은 정말 조심해서 사용해야 한다.
다른 사람이 실제로 lock을 들고 있는 상태에서 강제로 풀면 다시 다중 writer 문제가 생긴다.

Force unlock은 자동 unlock이 실패했고, 그 lock이 내 작업에서 생긴 것임을 확신할 때만 사용해야 한다.

---

## Backend별로 달라지는 운영 특성

Backend를 고르는 일은 "어느 클라우드를 쓰는가"만의 문제가 아니다.
어떤 잠금 모델을 쓸 수 있는지, 복구 전략이 있는지, 인증을 어떻게 다룰 수 있는지가 함께 결정된다.

| Backend | State 저장 위치 | Locking | 주로 생각할 점 |
| --- | --- | --- | --- |
| `local` | 로컬 파일 | 시스템 API 기반 지원 | 개인 실습에는 쉽지만 팀 운영에는 부적합 |
| `azurerm` | Azure Blob Storage | Azure Blob Storage 기능으로 지원 | Microsoft Entra ID 인증이 권장되며, storage container 권한 설계가 필요 |
| `consul` | Consul KV path | 지원 | Consul token, KV 권한, session 권한 관리가 중요 |
| `cos` | Tencent Cloud Object Storage bucket/prefix | 지원 | bucket versioning을 켜서 실수 삭제나 오류 복구 가능성을 확보하는 것이 좋음 |
| `http` | REST endpoint | 선택 지원 | GET/POST/DELETE와 선택적 LOCK/UNLOCK을 구현하는 자체 state service에 적합 |
| `kubernetes` | Kubernetes Secret | Lease 리소스로 지원 | Secret 읽기/쓰기 권한과 namespace, kubeconfig 또는 in-cluster 인증 설계가 필요 |
| `s3` | S3 bucket/key | `use_lockfile = true`로 S3 lockfile 지원 | bucket versioning 권장, lockfile 사용 시 `.tflock` 객체 권한 필요 |

특히 S3 backend는 오래전부터 DynamoDB locking 예시로 많이 알려져 있지만,
현재 문서 기준으로 DynamoDB 기반 locking은 deprecated이며 향후 minor version에서 제거될 예정이다.
새 구성에서는 S3 lockfile 방식인 `use_lockfile = true`를 우선 고려하는 것이 맞다.

---

## State를 나누는 일은 구조 변경이 아니라 소유권 변경이다

인프라가 커지면 state도 커진다.
State가 커질수록 plan/apply 시간이 길어지고, 작은 변경이 넓은 범위에 영향을 줄 가능성도 커진다.
이때 state refactor를 고려한다.

하지만 state를 나누는 일은 단순히 파일을 쪼개는 작업이 아니다.
Terraform 문서가 강조하는 기준은 운영 단위에 가깝다.

- 변경 주기가 다른가?
  - 예: 네트워크는 몇 달 동안 안정적인데 compute는 하루에도 여러 번 scale될 수 있다.
- stateful 리소스와 stateless 리소스가 섞여 있는가?
  - 예: database와 compute를 함께 두면 compute 재생성 작업의 폭발 반경이 database까지 번질 수 있다.
- 팀의 책임 경계가 다른가?
  - 예: platform 팀이 VPC를 관리하고 application 팀이 service를 관리한다면 state도 그 경계를 반영하는 편이 낫다.
- module로 재사용할 수 있는 논리적 묶음인가?

즉 좋은 state 분리는 Terraform 성능만을 위한 것이 아니라,
변경 권한과 장애 반경을 더 명확히 하는 작업이다.

---

## State를 나누면 dependency를 다시 설계해야 한다

하나의 state 안에서는 리소스 간 참조가 쉽다.

```hcl
subnet_id = aws_subnet.private.id
```

하지만 VPC를 다른 state로 옮기면 이 직접 참조는 더 이상 가능하지 않다.
그래서 state refactor 전에 dependency를 식별해야 한다.

문서에서 권장하는 방식은 hard-coding이 아니라 dynamic reference다.

- provider가 제공하는 data source로 원격 리소스를 조회한다.
- HCP Terraform/Terraform Enterprise에서는 `tfe_outputs`로 다른 workspace output을 참조한다.
- 다른 remote backend나 local backend에서는 `terraform_remote_state` data source를 사용할 수 있다.

다만 `terraform_remote_state`는 편리한 만큼 state output을 공유하는 방식이므로,
어떤 값이 외부 configuration에 노출되는지 신중하게 정해야 한다.

의존성 파악에는 `terraform graph`도 도움이 된다.
State를 쪼개기 전에 현재 리소스들이 어떤 방향으로 연결되어 있는지 먼저 시각화하면,
잘못 나눠서 서로를 강하게 물고 있는 구조를 만들 가능성을 줄일 수 있다.

---

## Stateful 리소스는 지우고 다시 만들면 안 된다

State refactor에서 가장 조심해야 하는 리소스는 database, object storage 같은 stateful 리소스다.
Stateless 리소스는 downtime이나 비용 문제가 없다면 새 configuration에서 다시 만들 수도 있다.
하지만 stateful 리소스는 삭제와 재생성이 곧 데이터 손실일 수 있다.

이 경우 Terraform state 상의 소유권만 옮겨야 한다.

현재 권장되는 방식은 `removed` block과 `import` block을 함께 쓰는 configuration-driven migration이다.

Source configuration에서는 리소스 블록을 제거하는 대신 `removed` block을 둔다.

```hcl
removed {
  from = aws_instance.example

  lifecycle {
    destroy = false
  }
}
```

이렇게 하면 Terraform은 해당 리소스를 더 이상 source state에서 관리하지 않지만,
실제 인프라는 destroy하지 않는다.

Destination configuration에서는 동일한 실제 객체를 새 리소스 주소로 import한다.

```hcl
resource "aws_instance" "example" {
  instance_type = "t3.micro"
  ami           = data.aws_ami.example.id
}

import {
  id = "i-07b510cff5f79af00"
  to = aws_instance.example
}
```

이 방식의 장점은 migration 의도가 코드에 남는다는 것이다.
반대로 `terraform state mv`로 state 파일을 직접 옮기는 방식도 있지만,
문서에서는 이를 legacy command로 설명하고 새 migration에는 `removed`/`import` block 방식을 권장한다.

어떤 방식을 쓰든 공통 원칙은 같다.

- 시작 전에 `terraform state pull`로 백업한다.
- `terraform plan`에서 destroy가 없는지 확인한다.
- source와 destination 양쪽에서 plan 결과가 의도와 맞는지 확인한다.
- remote state를 수동으로 `push`하는 작업은 가능한 피하고, 필요하다면 매우 조심한다.

---

## State pull/push는 응급 도구에 가깝다

Remote backend를 사용해도 `terraform state pull`로 state를 가져올 수 있고,
`terraform state push`로 state를 다시 밀어 넣을 수 있다.

하지만 `push`는 remote state를 덮어쓰는 작업이라 위험하다.
Terraform은 이를 막기 위해 두 가지 안전장치를 둔다.

- lineage가 다르면 다른 시점에 만들어진 state로 보고 허용하지 않는다.
- destination state의 serial이 더 높으면 더 최신 변경이 있다고 보고 허용하지 않는다.

`-force`로 이 보호를 우회할 수는 있지만,
그 전에 반드시 `terraform state pull`로 백업을 떠야 한다.

일반적인 운영에서는 state를 직접 편집하거나 push하는 흐름을 표준 절차로 만들지 않는 편이 좋다.
State는 Terraform이 관리해야 하는 데이터베이스이지, 사람이 자주 열어 고치는 설정 파일이 아니다.

---

## 정리: State 운영 원칙

Terraform state를 잘 다룬다는 것은 backend 종류를 외우는 것이 아니라,
다음 원칙을 지키는 일에 가깝다.

1. State는 실제 인프라와 Terraform 설정을 연결하는 데이터베이스다.
2. 팀 환경에서는 local state가 아니라 공유 remote backend를 사용한다.
3. Backend는 저장소, locking, 인증, 복구 전략을 함께 결정하는 운영 선택이다.
4. Credential은 backend 설정에 직접 쓰지 않고 환경 변수나 안전한 credential mechanism으로 주입한다.
5. Locking을 끄지 않는다. Force unlock은 내 lock이 자동 해제되지 않았을 때만 쓴다.
6. State를 나눌 때는 변경 주기, stateful/stateless 여부, 팀 소유권, 의존성을 먼저 본다.
7. Stateful 리소스 migration은 destroy/recreate가 아니라 `removed`/`import`로 소유권을 옮긴다.
8. `terraform state push`는 일반 작업 도구가 아니라 복구나 수동 수정이 필요한 예외 상황의 도구로 본다.

State는 Terraform의 부산물이 아니라 Terraform이 세상을 기억하는 방식이다.
그래서 state를 어디에 두고, 누가 쓰고, 어떻게 잠그고, 어떻게 나눌지 결정하는 일은 곧 Terraform 운영 설계를 결정하는 일이다.

---

## 참고 문서

- [Purpose of Terraform State](https://developer.hashicorp.com/terraform/language/state/purpose)
- [State Storage and Locking](https://developer.hashicorp.com/terraform/language/state/backends)
- [Refactor Terraform state](https://developer.hashicorp.com/terraform/language/state/refactor)
- [State Locking](https://developer.hashicorp.com/terraform/language/state/locking)
- [Backend block configuration overview](https://developer.hashicorp.com/terraform/language/backend)
- [Backend Type: local](https://developer.hashicorp.com/terraform/language/backend/local)
- [Backend Type: azurerm](https://developer.hashicorp.com/terraform/language/backend/azurerm)
- [Backend Type: consul](https://developer.hashicorp.com/terraform/language/backend/consul)
- [Backend Type: cos](https://developer.hashicorp.com/terraform/language/backend/cos)
- [Backend Type: http](https://developer.hashicorp.com/terraform/language/backend/http)
- [Backend Type: kubernetes](https://developer.hashicorp.com/terraform/language/backend/kubernetes)
- [Backend Type: s3](https://developer.hashicorp.com/terraform/language/backend/s3)
