## Networking

### CNI 란?
사전적 정의 "다양한 네트워크 구현체가 Kubernetes에 연결할 수 있도록 하는 표준 API"이다.
쿠버네티스 docs에서도 이야기했던 쿠버네티스 네트워크의 특징을 다시 한번 짚어보자.
> 1.모든 포드는 고유한 IP 주소를 갖습니다.
2.어떤 노드의 Pod든 NAT 없이 다른 모든 노드의 모든 Pod와 통신할 수 있습니다.

위와 같은 쿠버네티스 네트워킹을 가능케 하는 것에 중심에 CNI가 있다.
CNI 는 바이너리 배열을 기준으로 IPAM 플러그인과 네트워크 플러그인을 분리한다.


#### CNI IPAM 플러그인

IPAM 플러그인은 오직 "어떤 IP를 이 Pod에 줄 것인가"를 결정한다. 커널을 건드리지 않는다.  API server에 API 요청을 보내서 etcd/CRD를 읽고 쓰는 작업이 수행된다. 결과물은 IP 주소 하나와 게이트웨이 주소뿐이다. 즉 네트워크 플러그인이 직접 커널 레벨에 관여해서 파드를 쿠버네티스 네트워크에 등록하기 위한 지침을 만든다고 이해하였다.

![](https://velog.velcdn.com/images/victor1919/post/cc5e2ce3-78cd-4817-86f6-f0b9c306c9e8/image.png)


#### CNI 네트워크 플러그인

IPAM 플러그인에서 최종적으로 배열에 쓰기를 한 인덱스의 IP를 받아서 실제 커널 오브젝트를 만든다. veth 생성, Pod ns 내부 주소·라우팅 설정, host ns 라우팅 주입, proxy_arp 활성화, Felix를 통한(Network Policy watch) iptables/BPF 규칙 삽입까지가 이 플러그인의 책임 구간으로 IPAMBlock 을 기반으로 실제 OS  커널 네트워크 테이블을 수정해서 pod가 kubernetes 네트워크에 배포될수 있는 기반을 만든다.


![](https://velog.velcdn.com/images/victor1919/post/b542bd62-3998-4095-814e-b84dff084d22/image.png)

**1단계: Control-Plane (Pod 스케줄링)**

a. 사용자가 kubectl을 통해 Pod 생성을 요청하면 API-Server가 이를 수신 ->

b. API-Server RBAC 검사(인증 및 인가)와 어드미션 컨트롤(Admission Control)을 수행

c. etcd 저장 (Pending == pod status) 검증을 통과한 Pod 정보는 etcd에 Pending 상태로 저장됨.

d. Kube-scheduler의 Node 할당:

API-Server를 watch하고 있던 Kube-scheduler가 할당되지 않은 Pod(nodeName == empty)를 발견
Filtering, Scoring 단계를 거쳐 해당 Pod가 배치될 최적의 Worker-Node를 결정(Binding).

e. etcd 업데이트:

스케줄러가 결정한 Node 정보(node name = worker-node)가 API-Server를 통해 다시 etcd에 기록

**2단계: Worker-Node (Pod 생성 준비)**

a. Kubelet의 Pod Spec 수신:

할당된 Worker-Node의 Kubelet은 API-Server를 watch하다가 자신에게 할당된 Pod Spec을 pull.

b. Container Runtime 호출:

Kubelet은 Container Runtime에게 컨테이너 생성을 요청하고, Linux Namespace가 생성

**3단계: CNI & 네트워크 설정 (Calico)**

CNI 실행:

Kubelet은 CNI 설정 파일(/etc/cni/net.d/10-calico.conflists)을 읽어 CNI 플러그인을 실행(exec). 이 과정은 크게 IPAM 플러그인과 Network 플러그인 두 단계로 분할.

**3-1. IPAM Plugin (IP 할당 과정)**

Pod에 부여할 IP를 할당하기 위해 API-Server와 통신. (최소 3번의 API 요청 발생)

a. GET BlockAffinity: API-Server에 IPAM Block 정보를 요청하여 받아옴.

b. empty index 선택: 할당받은 Block 내에서 비어있는 IP 인덱스를 선택.

c. PUT allocations[n]: 선택한 IP 할당 정보를 API-Server에 업데이트.

d. POST IPAM Handle: IPAM 핸들 정보를 POST 요청으로 기록하고 IPAM 과정을 종료.

**3-2. Network Plugin (네트워크 인터페이스 구성)**

실제 네트워크 통신이 가능하도록 인터페이스와 라우팅 규칙을 설정.

a. veth pair 생성: Host와 Pod Namespace를 연결할 가상 이더넷 파이프(veth pair)를 생성

b. Pod ns 내부 설정: Pod의 네트워크 네임스페이스 내부에 할당받은 IP 주소를 설정하고 인터페이스와 라우팅을 활성 (ip addr add / link setup / route add).

c. Host 라우팅 주입: Host 쪽에 Pod로 향하는 라우팅 정보(Pod <-> Host)를 주입.

d. Felix 정책 적용: iptables chain or BPF MAP을 이용하여 Calico의 보안 및 네트워크 정책(Felix)을 적용.

e. CNI 종료: 모든 설정이 완료되면 JSON 형식의 결과를 출력하며 정상 종료(exit 0).

**4단계: Pod Running**
CNI의 네트워크 구성이 성공적으로 끝나면, 컨테이너가 정상적으로 실행되며 Pod는 Running 상태 전환. 이 상태 정보는 다시 Kubelet을 통해 API-Server로 전달된다.



### 네트워크 모드
Calico 의 네트워크 구성방식에는 Overlay Network 모드(L3)인 VXLAN , Ip in IP와 Underlay Network Mode(L2)인 BGP 모드가 있다.

#### CrossSubnet
기본적으로 크로스 서브넷이란, Kubernetes 클러스터의 노드들이 같은 L2 세그먼트(같은 스위치)에 있지 않은 상황을 이야기한다. 이 상황을 해결하기 위해서 나온 것이 오버레이 네트워크이고 Calico 에서는 아래의 2가지 오버레이 네트워크 방식을 지원한다. Calico 네트워크에서 해당 모드를 키게 되면 동일 스위치로 묶여있는 L2 안에서는 Overlay 없이 Native 라우팅을 하고 그렇지 않다면 Overlay 라우팅을 하게된다.


![](https://velog.velcdn.com/images/victor1919/post/60d628fd-999b-49f8-8ac0-15bc33ae3ffc/image.png)
#### IP in IP
이름 그대로** IP 패킷 안에 IP 패킷**을 넣는 방식이다.
원본 패킷에 노드  ip 헤더를 추가하여 감싸는 방식이다. 노드의 ip 헤더가 추가되기 때문에 L3 패킷이 되며 20바이트의 오버헤드가 생기게 된다.  VXLAN 과 달리 UDP를 쓰지 않기 때문에 포트 기반 로드밸런싱이 안 된다는 점이 있다.

#### VXLAN
UDP로 감싸는 방식이다. L2 프레임 전체를 UDP 패킷 안에 넣는다.
추가되는 헤더 크기는 50바이트로 IP-in-IP보다 30바이트 더 크다.
UDP를 쓰기 때문에 포트를 기반으로 로드밸런싱이 가능하다.BGP가 전혀 없어도 되기 때문에 클라우드 환경이나 단순한 인프라에서 효용성이 높다.




파드 IP 주소가 클러스터 외부의 더 넓은 네트워크로 라우팅될 수 있는지 여부
라우팅 가능
Snat 을 통해서 ip 변환 -> pod는 절대 snat 변환 사실을 모른다
라우팅 불가능

#### BGP
Calico 의 네트워크 모드에서 BGP 모드가 있지만 레거시 온프레미스에서 부터 존재하던 BGP는 토폴로지 네트워킹 방식이다. 전통적인 BGP의 개념에 대해서 알아보도록하자.

![](https://velog.velcdn.com/images/victor1919/post/b5e1606f-2cb1-4f3b-94f9-4a639534a803/image.png)

**Boarder Gateway Protocol**

AS(Autonomous System) 사이의 경로 지정(라우팅)을 위해 사용되는 프로토콜이다.
여기서 AS란?
**AS(Autonomous System, 자율 시스템)**
하나의 단위로 관리되는 네트워크 집합을 의미한다. 예를 들어서 kt에서 관리하는 네트워크망은
KT AS일 것이고, AWS에서 운영하는 네트워크망은 AWS AS 일 것이다.
각 AS는 고유한 번호가 주어지며, 이를 ASN(Autonomous System Number)라고 한다.
여기서 서로 다른 AS 끼리 통신하기 위한 프로토콜이 BGP이며, iBGP와 eBGP로 나뉘게 된다.

**iBGP**
서로 같은 AS 상의 Border Gateway들 끼리의 연결을 담당하는 BGP
**eBGP**
서로 다른 AS 상의 Border Gateway들 끼리의 연결을 담당하는 BGP

BGP는 peering 이라는 메커니즘을 사용하여 작동한다. 관리자는 특정 라우터를 BGP 피어 또는 BGP 스피커 라우터로 할당한다. 피어는 자율 시스템의 엣지 또는 경계에 있는 디바이스라고 이해하였다.

**경로 검색**
BGP 피어는 Network Layer Reachability Information(NLRI) 및 경로 속성을 통해 인접 BGP 피어와 라우팅 정보를 교환한다. NLRI에는 인접 피어에 대한 연결 정보가 포함됩니다. 경로 속성에는 지연 시간, 홉 수, 전송 비용 등의 정보가 포함된다.

정보를 교환한 후 각 BGP 피어는 주변 네트워크 연결의 그래프를 구성할 수 있다.

**경로 저장**
검색 프로세스 중에 모든 BGP 라우터는 경로 알림 정보를 수집하여 라우팅 테이블 형태로 저장한다. 라우팅 테이블을 사용하여 경로를 선택하고 자주 업데이트한다.

예를 들어 BGP 라우터는 30초마다 인접 라우터로부터 연결 유지 메시지를 수신한다. 그리고 그 메시지에 따라 저장된 경로를 업데이트한다.

**경로 선택**
BGP 라우터는 저장된 정보를 사용하여 트래픽을 최적으로 라우팅한다. 경로 선택의 주된 요소는 저장된 경로 그래프에 의해 결정되는 최단 경로로, 여러 경로에서 대상에 도달할 수 있는 경우 BGP는 다른 경로 속성을 순차적으로 평가하여 최상의 대상을 선택하게된다.

[https://aws.amazon.com/ko/what-is/border-gateway-protocol/](https://aws.amazon.com/ko/what-is/border-gateway-protocol/)

#### Calico 로 구현되는 BGP

![](https://velog.velcdn.com/images/victor1919/post/d43f1eec-4e73-4623-82eb-eacc5845a591/image.png)
iBGP의 경우 컨트롤플레인이 Route Reflector가 되어 워커노드의 광고를 받고 중계해주는 역할을 한다.
eBGP에서는 TOR을 제외하면 모든 노드는 동일하게 광고해주고 Router와 직접 피어링을 맺어 토폴로지가 구축된다.
따라서 클러스터 A의 pod가 클러스터  B의 pod e 로 패킷을 전송하기 위해서 Router를 거치게 되고,
만약 동일 클러스터 내에서는 L2 안으로 통신이 이루어지는 것을 알 수 있다.




## Networking Policy

### Zero trust network model
제로 트러스트 네트워크 모델이란, 외부 네트워크가 전부 공격적이라고 가정하고 설계하는 네트워크 보안 모델을 이야기한다.  또한 신뢰할 수있는 진입자라 하여도 반드시 검증 절차를 거친다.

Calico 에서는 해당 모델을 구현하기 위한 5가지의 요구사항을 제시하고 있다.

**요구사항 1**
모든 네트워크 연결은 (단순히 특정 영역의 경계를 넘나드는 연결뿐만 아니라) 예외 없이 정책 제어 및 집행의 대상이 된다.

**요구사항 2**
원격 엔드포인트(단말)의 신원을 확인 할 때는 항상 강력한 암호화 기반의 신원 증명을 포함한 다각적인 기준을 바탕으로 해야 한다. 특히 IP 주소나 포트 번호 같은 네트워크 수준의 식별자는 적대적인 네트워크 환경에서 위·변조될 수 있으므로, 단독으로는 신원 확인 기준으로서 충분하지 않다.

**요구사항 3**
예상되고 허용된 모든 네트워크 흐름(Flow)은 명시적으로 허용되어야 한다. 명시적으로 허용되지 않은 모든 연결은 기본적으로 차단(거부)된다.

**요구사항 4**
침해된(해킹당한) 워크로드가 보안 정책의 제어와 집행을 우회할 수 있어서는 안 된다.

**요구사항 5**
은 제로 트러스트 네트워크는 적대적인 존재가 네트워크 트래픽을 도청하여 민감한 데이터를 유출하는 것을 막기 위해 트래픽 암호화에 의존한다. 네트워크를 통해 개인 데이터가 교환되지 않는다면 이것이 절대적인 요구사항은 아니지만, 제로 트러스트 네트워크의 기준에 부합하려면 암호화가 필요한 모든 네트워크 연결에서 예외 없이 암호화가 사용되어야 한다. 제로 트러스트 네트워크는 '신뢰할 수 있는 네트워크 경로'와 '신뢰할 수 없는 네트워크 경로'를 구분하지 않는다. 또한, 데이터 프라이버시를 위해 암호화를 사용하지 않는 경우라 하더라도 신원을 확인하기 위한 암호학적 인증 증명은 여전히 사용된다는 점에 유의해야 한다.

### Tier

![](https://velog.velcdn.com/images/victor1919/post/d072bca2-4724-485e-8e73-d9907e08cc30/image.png)

"티어"를 통해서 네트워크 정책간의 우선순위를 정할 수 있다.
예를 들어서 보안팀 > 플랫폼 팀 > 개발 팀 순으로 티어를 나눌 수 있다.

![](https://velog.velcdn.com/images/victor1919/post/b18272b6-52d4-4ec4-b5f4-8c1987fca536/image.png)

이러한 계층형 네트워크 정책에서 중요한 기능은 Pass라고 한다.
Pass는 해당하는 정책을 allow 하지도 deny 하지도 않고 다음 계층으로 넘긴다는 뜻이라고 한다. 위 flow를 이해해 보자면

Tier 2
policy A, B, C, D가 순서대로 평가된다. policy D에 도달했을 때 Pass 액션이 발동된다. Tier 2의 Default deny는 실행되지 않는다. Pass가 명시적으로 다음 Tier 평가를 지시했기 때문이다.
Tier 3
X 표시가 쳐져 있다. 이 Tier의 policy E, F, G, H 중 이 트래픽의 엔드포인트와 일치하는 selector를 가진 정책이 하나도 없다는 뜻이다. . Default deny도 실행되지 않는다. 일치하는 정책이 없으면 묵시적 거부도 발동하지 않는 것이 Calico Tier의 규칙이다.
Tier 4
policy I를 평가한다. 일치하지 않아 넘어간다. policy J에서 이 트래픽의 엔드포인트와 일치하는 selector가 발견된다. 화살표가 policy J를 가리키는 이유다. 여기서 Allow 또는 Deny가 최종 결정된다.

핵심 규칙 두 가지
규칙 1 — Pass는 현재 Tier의 나머지 정책과 Default deny를 건너뛰고 다음 Tier로 평가를 넘긴다. Tier 2에서 policy D 아래에 있는 Default deny가 실행되지 않는 이유다.
규칙 2 — 어떤 Tier에 해당 엔드포인트와 일치하는 정책이 단 하나도 없으면 그 Tier 전체를 건너뛴다. Default deny조차 실행되지 않는다. Tier 3이 X로 표시되고 완전히 스킵되는 이유다.

만약 Tier 3에 일치하는 정책이 있었다면
Tier 3의 어느 정책이라도 이 엔드포인트와 일치했다면 Tier 3에서 평가가 진행됐을 것이다. 그 정책이 Deny를 내리면 Tier 4는 평가되지 않고 트래픽이 차단됐을 것이다. Pass를 내렸다면 Tier 4로 넘어왔을 것이다. Tier 3 끝까지 갔는데 명시적 Allow/Deny/Pass가 없었다면 Tier 3의 Default deny가 발동해서 차단됐을 것이다.

### preDNAT 과 applyonForward가 세트인 이유

**preDNAT**은 네트워크 정책의 옵션 중 하나로, 패킷이 들어왔을 때, 네트워크 정책이 적용되는 시점을 목적지 ip 변환 작업인 DNAT이 적용되기전 (노드의 ip를 pod의 ip로 변환하기전) 으로 잡는 것이다. 네트워크 정책의 적용 타이밍이 FORWARD 훅이기 때문에 (Netfilter에서) PREROUTING 훅에서 DNAT 이 발생하기 때문에 존재한다.

**applyonForward** Calico 정책은 기본적으로 엔드포인트 에만 적용된다. 즉 파드가 목적지인 경우 노드를 통과하는 포워딩 트래픽에는 적용되지 않는다.
포워딩 트래픽이란 노드가 최종 목적지가 아닌 패킷이다. 이 패킷은 노드의 host 엔드포인트를 통과하지만 Pod veth는 통과하지 않는다. 만약 applyOnForward를 켜면 이런 포워딩 트래픽에도 네트워크 정책이 적용된다.

둘을 반드시 같이 켜야 하는 이유는 preDNAT을 켜도 applyonForward를 키지 않는다면 DNAT 이 적용되기 전이라면 포워딩 트래픽에 해당되기 때문에 네트워크 정책이 적용되지 않아 네트워크 정책이 적용되지 않는다. 즉 아무 효용이 없다.

### Calico doNotTrack 네트워크 정책

doNotTrack은 conntrack 을 완전히 건너뛰도록 하는 설정이다.
기본적으로 iptables의 conntrack은 패킷의 모든 상태를 추적한다.
하지만 대규모 환경의 경우 conntrack이 소모하는 cpu와 메모리를 극한으로 줄이기 위해
doNotTrack을 통한 conntrack 스킵으로 CPU 사이클과 메모리 접근을 줄일 수 있다.
eBPF를 사용한다면 eBPF의 사용이 더 효과적인 방법이겠지만 iptables를 사용하는 환경에서는 괜찮은 선택이 될 수 있다.
일반적인 conntrack을 거치는 패킷과 doNotTrack이 적용된 패킷의 흐름을 비교해보자.
![](https://velog.velcdn.com/images/victor1919/post/8159cc60-64cf-4581-a22b-343227cb2d56/image.png)

일반적으로 새 패킷이 들어오면 conntrack에서 새로운 추적을 생성하고 iptables의 규칙에 따라 필터링하여 서버로 패킷이 들어가게된다.
하지만 doNotTrack 옵션을 키게 되면 넷필터의의 raw table에 UNTRACKED 라는 마킹을 인지하고 conntrack을 skip 하게 된다. 이후 Calico 의 Felixt가 생성한 정책에 의해 filter에서 필터링되어 서버로 패킷이 들어간다.