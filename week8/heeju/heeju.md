## 쿠버네티스 확장

**오퍼레이터(Operator) 패턴**
: 특정 애플리케이션 운영 로직을 자동화하는 Kubernetes 확장 패턴  
-> 사람이 하던 운영 작업을 자동화한 컨트롤러

Operator = CRD + Controller 조합

**CRD (Custom Resource Definition)**  
: 새로운 리소스 타입 정의

**Controller**  
: 계속 상태 감시함 (Control Loop)

<pre>Desired State → Current State 비교 → 맞추기</pre>

<배포 방식>

1. CRD 등록
2. Controller를 Deployment로 실행

#### 일반 Controller vs Operator 차이?

- Controller
  - 리소스 상태 맞추는 일반 로직
- Operator
  - 특정 앱 운영 지식까지 포함한 고급 Controller

=> Operator ⊃ Controller

**쿠버네티스 네트워크 플러그인 (CNI, Container Network Interface)**
Pod 네트워크를 실제로 만들어주는 플러그인  
-> Pod한테 IP 주고 서로 통신 가능하게 만듦

쿠버네티스는 네트워크 기능을 직접 구현하지 않고 CNI에 맡긴다! CNI가 쿠버네티스 네트워크의 실질 구현체임

<흐름>

<pre>
Pod 생성
 ↓
kubelet
 ↓
Container Runtime (containerd / CRI-O)
 ↓
CNI Plugin 실행
 ↓
IP 할당 + 인터페이스 생성 + 라우팅 설정
</pre>

-> kubelet이 직접 CNI 관리 안 함 (Runtime 책임)

대표 CNI들: Calico, Flannel, Cilium 등

**장치 플러그인 (Device Plugin)**  
: GPU, FPGA, 특수 NIC 같은 하드웨어를 Kubernetes에서 쓰게 해주는 확장 기능  
-> 쿠버네티스가 모르는 하드웨어를 등록해서 쓸 수 있게 해줌

기본 Kubernetes 리소스(CPU, Memory, Storage)는 기본으로 지원하지만, GPU, FPGA 같은 특수 장치는 vendor 설정이 필요함!!

<동작 흐름>

<pre>
Vendor Device Plugin
        ↓
kubelet 등록
        ↓
Node Status 업데이트
        ↓
API Server에 리소스 노출
        ↓
Pod가 요청
</pre>

Device Plugin은 kubelet에 gRPC로 등록함 (등록 정보: Unix Socket 이름, API 버전, ResourceName)
