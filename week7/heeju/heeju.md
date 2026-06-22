## 클러스터 관리

**그레이스풀(Graceful) 노드 셧다운(shutdown)**  
: 노드가 꺼지기 전에 미리 알고 정상 종료하는 방식

<흐름>

<pre>
shutdown 감지
↓
노드 NotReady
↓
새 파드 스케줄 금지
↓
기존 파드 순서대로 종료
↓
노드 종료
</pre>

노드가 꺼질 예정이니까 새 파드 보내지 말라고 스케줄러한테 인식시키고 기존 파드들을 종료하고 노드를 종료함

**논 그레이스풀 노드 셧다운**  
: 정전이나 기타 외부 요인과 같은 이유로 예기치 않게 노드가 셧다운되는 것  
->

<흐름>

<pre>
노드 갑자기 죽음
↓
kubelet 반응 못 함 (파드나 볼륨 정리 불가능)
↓
Pod terminating 고착 (삭제가 안 됨 ㅠㅜ)
↓
Volume detach 안 됨
↓
다른 노드에 복구 불가
</pre>

<해결 방법>

<pre>kubectl taint node node1 node.kubernetes.io/out-of-service=:NoExecute</pre>

-> 파드 강제 삭제, 볼륨 즉시 detach 해서 다른 노드에 재생성 가능!  
노드가 완전히 죽은 걸 확인 후에 실행해야 함... 안 그러면 데이터가 꼬일 수 있다고 함

**드레인(drain)**  
: 노드 종료 전 노드를 비우는 작업 (재부팅, OS 업데이트, 하드웨어 교체 등 할 때)

<pre>kubectl drain node1</pre>

<순서>

1. cordon (더 이상 이 노드에 새 파드 스케줄 금지)
2. eviction (기존 파드들을 하나씩 내보냄(삭제 후 다른 노드에 재생성))

**노드 오토스케일링 (Node Autoscaling)**  
: 파드 수에 따라 노드 개수를 자동으로 늘리거나 줄이는 기능

-> 파드가 많아져서 자리가 부족하면 노드를 늘리고, 파드가 줄어서 남는 노드가 생기면 노드를 줄여서 비용을 아낌

**노드 프로비저닝 (Provisioning)**  
: 새로운 노드를 만드는 것

<흐름>

<pre>
Pod 생성
↓
기존 노드에 자리 없음
↓
Pending 상태
↓
오토스케일러 감지로 새 노드 생성
↓
Pod 실행
</pre>

-> Pending 파드가 생기면 새 노드를 자동 생성함 (**Pending**: 스케줄링 못 된 상태)

**노드 통합 (Consolidation)**  
: 필요 없는 노드를 줄이는 것 (안 쓰는 노드를 정리해서 비용 절약)

트래픽이 감소되면 Pod 개수가 감소해서 빈 노드 생김  
-> 오토스케일러가 감지해서 노드 제거

**Cluster Autoscaler**  
: 미리 정해둔 노드 그룹 안에서만 노드를 증감하는 오토스케일러

**Karpenter**
: 필요한 노드를 즉석에서 생성하는 오토스케일러
ex)

<pre>
CPU 8 필요한 Pod 발견
↓
거기에 맞는 VM 자동 선택
↓
노드 생성
</pre>

**인증서 (Certificate)**  
: 통신하는 상대가 진짜 맞는지 확인하는 신분증 같은 것  
-> 쿠버네티스에서는 주로 API Server, kubelet, 사용자(kubectl) 사이 통신을 안전하게 하기 위해 사용

**CA (Certificate Authority)**  
: 인증서를 발급하는 기관

**SAN (Subject Alternative Name)**  
: 이 인증서가 허용하는 주소 목록
-> 목록에 주소가 없으면 오류 뜸

**로깅 아키텍처**  
: 애플리케이션이 실행되면서 남기는 기록들  
-> 장애 분석, 디버깅, 모니터링할 때 씀

로그 확인: `kubectl logs pod이름`

**로그 로테이션 (rotation)**  
: 로그 파일이 너무 커지면 자동으로 잘라서 새 파일로 만드는 것

**클러스터 레벨 로깅**  
: 노드가 죽으면 로그도 날아갈 수 있기 때문에 로그를 외부 저장소에 따로 모으는 것

-> 쿠버네티스 기본 제공 X 직접 구축해야 함

### 로그 수집 방법

1. 노드 레벨 에이전트
   - 각 노드마다 로그 수집기 하나씩 실행
   - 앱 수정 필요 없이 관리 쉬움
2. Sidecar 방식
   - Pod 안에 로그 전용 컨테이너 추가
   - 로그 분리 쉽지만 컨테이너 수 증가
3. 앱이 직접 외부로 전송
   - 빠르지만 앱 코드 수정 필요

<pre>
프로덕션 환경에서는 이러한 메트릭을 주기적으로 수집하고 시계열 데이터베이스에서 사용할 수 있도록 프로메테우스 서버 또는 다른 메트릭 수집기(scraper)를 구성할 수 있다.
</pre>

**프로메테우스(Prometheus)**  
: 메트릭 수집기 + 시계열 데이터베이스(Time Series DB)
-> 서버들 상태 숫자(메트릭)를 주기적으로 스크랩해서 저장

#### 프로메테우스가 하는 일

1. 스크랩
   - 주기적으로 매트릭 가져옴(15초마다, 30초마다... 등)
2. 저장
   - `metric_name{label=value} number` 이런 식으로...
3. 조회
   - 프로메테우스 전용 쿼리 언어(PromQL) 사용
   - ex. 최근 5분 평균 CPU 점유율 `rate(container_cpu_usage_seconds_total[5m])`
4. 알림
   - 내가 정한 조건을 만족하면 알림 발생
   - Slack, Discord, Email 알림 가능

**에뮬레이션 버전 (Emulated Version)**  
: 실제 바이너리 버전보다 낮은 버전처럼 동작하게 만드는 기능
-> 설치된 프로그램 버전이 최신 버전이어도 이전 버전처럼 동작할 수도 있음

**kube-state-metrics**  
: 쿠버네티스 오브젝트 상태를 메트릭 형태로 뽑아주는 애드온

쿠버네티스 자체 컴포넌트(apiserver, kubelet)는 CPU/메모리 같은 시스템 메트릭을 뽑는데, kube-sate-metrics는 **클러스터 안 오브젝트가 지금 어떤 상태인지**를 메트릭으로 보여줌
-> 리소스 사용량이 아닌 쿠버네티스 상태 자체를 보여주는 용도

**Klog**  
: 쿠버네티스 기본 로깅 라이브러리
-> 쿠버네티스 핵심 컴포넌트들(kube-apiserver, kube-scheduler 등)이 로그 찍을 때 사용

**Structured Logging**  
: 로그를 key=value 형태로 구조화해서 찍음

**쿠버네티스 시스템 컴포넌트 추적 (Tracing)**  
: Kubernetes 내부 요청의 흐름과 병목 지점을 추적하는 기능
-> 로그(log)가 무슨 일이 일어났는지를 보여준다면, tracing은 **“어디서 얼마나 시간이 걸렸는지”**를 보여줌

<흐름>

<pre>
K8s Component → OTLP(gRPC) → Collector → Tracing Backend
</pre>

#### 쿠버네티스 프록시(Proxy)

1. kubectl proxy
   - 사용자가 직접 실행하는 로컬 프록시
   - 로컬에서 API 테스트 시 주로 사용
2. apiserver proxy
   - 클러스터 내부 리소스 접근 가능, Pod/Node/Service 연결 가능
3. kube-proxy
   - 각 노드에서 동작하는 Service 트래픽 프록시 (Service의 가상 IP를 실제 Pod로 연결)
   - 로드밸런싱 수행, TCP/UDP/SCTP 지원
4. LoadBalancer
   - 외부 트래픽 진입점
