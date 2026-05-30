### 클러스터 관리
쿠버네티스 실제 생성 및 사용 시의 관리에 대한 로우레벨 정보를 다룬다.

#### Admission Webhook

어드미션 웹훅이란 쿠버네티스 API 서버에 요청이 들어왔을 때, 그 요청이 최종적으로 클러스터에 반영되기 전에 가로채서 검사하거나 수정할 수 있게 해주는 확장 플러그인이다.
어드미션 웹훅에는 변경(Mutating) 과 검증 웹훅의 주요한 단계를 거치는데 이에 대해서 알아본다.

![](https://velog.velcdn.com/images/victor1919/post/3ea1f108-33b7-4eac-8fa3-604eb391eab9/image.png)

WebHook

1. 오브젝트 제출
   Monkey라는 이름의 CRD 오브젝트를 생성하고자 하고 있다.

2. API 요청
   당연히 생성된 요청이 쿠버네티스 API 서버 내부로 진입한다.
   API 서버 내에서 HTTP 핸들러거 먼저 요청을 받고 이후 요청을 보낸 사용자의 인증을 거치게 된다.

3. Mutating Admission 단계 진입
   인증을 통과하게 되면 오브젝트의 정보를 AdmissionReview라는 josn 규격으로 감싼뒤, 클러스터 외부의 별도로 떠 있는 웹훅 핸들러 서버로 발송이 일어나난다.

4. 웹훅 서버의 변환 패치
   웹훅 서버에서 요청을 검토한뒤 allowd:true 를 담아서 수정할 내용이 있다면 patch 필드에 담아서 보낸다.

5. mutating 된 오브젝트의 상류 전달 및 스키마의 검증
   Object Schema Validation 단계로 진입하여 웹훅에 의해 수정된 내용이 올바른 쿠버네티스 규격을 따르고 있는지 검증한다.

6. Validating Adimssion 단계
   검증 어드미션까지 통과한다면 데이터가 영구 저장된다.
   Mutating Admission 과 동일하게 웹훅 핸들러로 검증을 수행하게 되는데 해당 요청을 승인할지 거부할지만 결정하게된다.




#### 로그 아키텍쳐

서 쿠버네티스 환경에서 로그 스트림 및 통합 로그 시스템을 구축하는 방법은 이렇게 3가지가 있다고한다.

![](https://velog.velcdn.com/images/victor1919/post/a721b0ad-2ae0-4c90-94d6-94836e5cc8ea/image.png)

첫번째는 노드 로깅 에이전트를 사용하는 방법이다.
Loki나 ELK 스택 처럼 각 노드에 데몬셋으로 실행되어서 어플리케이션 파드의 로그들을 전부 수집하여 로깅백엔드로 푸시한다.

![](https://velog.velcdn.com/images/victor1919/post/d265c200-2d10-4f76-91b5-dcc6eb4527e0/image.png)
해당 케이스는 어플리케이션이 로그를 직접 stdout하지않고 파일로 저장할 경우에 사이드 카 컨테이너가 stdout 혹은 stderr 스트림으로 기록하게 하고 별도 올린 로깅에이전트가 이를 로깅백엔드로 푸시하는 것이다. 이 방법을 통해 2개이상의 사이드 카 컨테이너를 통해서 별도의 로그 수집도 가능하다.

![](https://velog.velcdn.com/images/victor1919/post/1e8c4118-028a-4905-ac3c-06e1b6163d52/image.png)

어플리케이션 자체의 로깅에이전트를 별도의 사이드카 컨테이너로 생성하여 로깅백엔드로 푸시한다. 다만 파드가 굉장히 무거워지므로 권장되지 않는다.


### 클러스터 확장
#### CRD와 AA

둘은 동일하게 기존의 쿠버네티스 클러스터에서 구축하는 인프라의 특성에 맞게 추가적인 확장을 할 수 있게끔하는 요소라는 점에서는 동일하다. 하지만 구현 방법에서 큰 차이를 보이게 된다.
**CRD(Custom Resource Definition)**는 kube-apiserver에 새로운 리소스의 스키마를 등록하는 명세서의 형태를 지니게 된다.
사용자가 CRD를 yaml로 정의를 하여 apply 하게 되면 CRD에 정의한 새로운 kind 타입의 리소스가 생성되게 된다. 어플리케이션의 자동 배포 및 지속적 모니터링 백업 복원등의 기능을 자동으로 수행하는 Operator 개발을 위해 함께 많이  쓰인다.
**AA**는 API Aggregation의 약자로, 기존 kube-apiserver 옆에 완전히 독립된 별도의 api 서버를 개발하여 클러스터에 등록하고 트래픽을 중계받는 형식이다. 즉 특정 api 경로로 들어오게 되는 요청을 직접 생성한 API 서버에서 처리할 수 있게 끔하는 것이다.
CRD는 쿠버네티스의 기본 etcd 데이터베이스를 공유해야하지만, AA는 별도의 데이터베이스를 가질 수 도 있다. 대부분의 환경에서 확장리소스가 필요할 경우 CRD를 선택하는 것이 일반적이며, AA의 경우 특수한 목적을 가지고 있는 경우에만 제한적으로 사용된다.

### Calico 개요
Calico는 Kubernetes의 CNI 이다.

#### Overlay Network
![](https://velog.velcdn.com/images/victor1919/post/61207806-07b5-41b0-bcdf-d03b08462166/image.png)

우선 용어를 짚고 넘어가보자.
언더레이 네트워크 : 실제 물리네트워크
오버레이 네트워크 : 실제 물리네트워크 위에 소프트웨어로 Overlay (겹쳐서) 만든 네트워크

Calico 에서는 오버레이 네트워크를 언더레이 네트워크가 오버레이 네트워크의 정보에 대해서 전혀 알지 못해도 네트워크 장치들이 언더레이 네트워크를 통해 통신할 수 있도록 해준다고 기술하고 있다.
여러 종류가 있겠지만 공통적으로는 내부패킷이라 불리우는 네트워크 패킷을 외부 패킷으로 캡슐화하는 공통적 특징을 가지고 있다는 것이다. VXLAN의 경우에 L2 패킷을 발송할때 원본 패킷에 VXLAN 헤더를 붙이고 이를 다시. UDP 헤더와 IP, Ethernet 헤더를 붙여서 나가게 된다. 이를 통해 나가는 패킷이 물리 장비가 이해할 수 있는 패킷으로 변환되게 되고 목적지에 도착하면서 이를 벗겨내면 다시 L2 패킷이 되는 것이다. 이 캡슐화라고 불리는 기술을 이용하여 오버레이 네트워크가 이루어진다.

MTU(Maximum transmission unit)는 오버레이 네트워크에서 중요한 단위이다. 네트워크 링크의 최대 전송 단위인 MTU는 TCP가 경로의 MTU를 학습하고 경로의 MTU 중 가장 작은 MTU 크기를 기준으로 패킷의 크기를 조정하게 된다. 만약 패킷이 더많은 데이터를 (MTU보다) 전송하려고 한다면 세그먼트 분할을 통해 MTU의 초과를 방지한다.

#### Network Policy
쿠버네트스 네트워크 모델은 기본적으로 모든 파드가 파드 ip를 통한다면 통신이 허용된 평면 네트워크이다. 하지만 이를 네트워크 정책을 통해 파드와 파드사이의 네트워크 보안이 이루어지게끔 한다.
#### k8s network basic & eBPF

****externalTrafficPolicy: Cluster**(기본)**
![](https://velog.velcdn.com/images/victor1919/post/781c2829-5e7e-4750-8e33-a1b2a3be8737/image.png)

원본 소스가 클라이언트 -> Node 1 로 변경되었는데 이는 프록시와 파드의 노드가 다르기 때문에 중간과정에 SNAT이 발생하게 된것이다.

**externalTrafficPolicy: Local**
![](https://velog.velcdn.com/images/victor1919/post/440c67d2-095a-448d-8fa0-a3fa5ab01b8a/image.png)
로드밸런서나 라우터의 목적지가 실제로 pod가 살아 있는 노드로만 꽂아주게 되어 SNAT이 발생하지 않고 hop 이 한번 감소한다.
하지만 로드밸런싱에 의해서 균등 부하되는 트래픽에서 한쪽 노드에 pod가 쏠려있을 경우 pod가 적게 스케쥴링된 노드에서는 과부하가 발생하게 된다.

![](https://velog.velcdn.com/images/victor1919/post/4ab4de71-6711-49e0-97cc-fef606c27f80/image.png)

만약 eBPF가 개입한다면? node1의 eBPF는 service1의 목적지가
Node2란느 걸 인지하고 있다. 이에 SNAT 없이 그대로 Node2로 전달할 수 있다. 이 때 DSR(Direct Server Return)으로 반환이 이루어지며 Node2 에서 원래 외부 클라이언트에게 직접 전달 한다. 쿠버네티스에서의 eBPF 도입에 대해 Calico는 초반부에 좀더 자세히 기술하고 있었다.

#### 노드 간의 바이패스 및 커넥션의 추적

![](https://velog.velcdn.com/images/victor1919/post/c6b78b56-9ee5-4433-ad34-dacb4c196288/image.png)

pod가 veth를 통과할때 Traffic Control BPF 이 작동하여 CT Map을 lookup 하여 이미 승인되었던 flow 면 bypass 하고 패킷에 Mark 표시를 하게 된다 그렇게 물리 인터페이스(eth)로 나가기 전에 물리인터페이스의 tcBPF에서 Mark를 확인하면 iptables 규칙을 스킵한다. 패킷을 수신하게된 Node2에서도 동일한 과정이 일어나게 된다.

#### tc BPF의 정책 최적화
![](https://velog.velcdn.com/images/victor1919/post/fd87c8e6-869b-4e76-a586-3ddb7a0c274f/image.png)
tcBPF의 동작을 좀 더 자세히 살펴보게 되면 물리인터페이스로 패킷이 들어왔을 때 연결이 확립된 트래픽은 복잡한 검사 없이 바로 veth로 쏘아주는 Fast-path를 태운다.
NetworkPolicy가 생성하면, Calico는 일일이 해석하는 대신 정책 선택기와 일치하는 IP 집합을 참조하여 최적화된 바이트 코드로 컴파일 하고 Policy program이 패킷을 검사하여 허용 여부를 판별하고 에필로그 프로그램을 거쳐서 안전하게 파드로 인계한다.

#### 소켓 계층의 로드 밸런싱

![](https://velog.velcdn.com/images/victor1919/post/11070401-c1e4-4217-8efe-667a6066b5f9/image.png)

위 아키텍쳐는 유저스페이스의 프로그램이 네트워크 연결을 시도할 때 커널 레이어에서의 최적화 단계를 보여준다.
Calico 의 BPF는 소켓 API 계층에 connect BPF Hook을 걸어서 프로그램이 connect() 시스템콜을 호출하는 순간 요청을 가로채서 NAT maps를 체크한다. 쿠버네티스 서비스 주소라면, 패킷 생성전에 소켓 레벨에서 목적지 ip 주소 를 백엔드 pod의 실제 ip 주소로 다이렉트 변환하여 연결을 맺는다.

