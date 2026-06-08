# Ceph Object Gateway (RGW)
## RGW
- Ceph Object Gateway는 librados 위에 구축된 오브젝트 스토리지 인터페이스로, 애플리케이션과 ceph 스토리지 클러스터 사이에 RESTful 게이트웨이를 제공함
  - librados: RADOS(Reliable Autonomic Distributed Object Store)와 통신하는 라이브러리
- RGW는 복잡한 분산 저장 계층(RADOS)을 추상화해 S3와 Openstack Swift API로 나타냄
  - 두 API 모두 공통 네임스페이스를 공유함
<img width="1196" height="782" alt="image" src="https://github.com/user-attachments/assets/b92a4b7d-79bb-425a-9df2-14162ab36310" />

- RGW는 리눅스 운영체제의 사용자(root, admin..)나 ceph 내부 인증을 쓰는 대신, s3 Access key 같은 자체적인 계정 DB를 가지고 사용자 인증을 처리함
- MDS(Metadata Server) 데몬을 사용하지 않음
  - 일반적 파일 시스템은 트리 구조(C 드라이브 -> 사용자 -> 내문서 -> 사진폴더 ..)
  - 특정 파일을 찾으려면 계층별 탐색이 필요 (디렉토리 경로, 권한, 파일 잠금 등에 관한 메타데이터를 관리하는게 MDS)
- RGW(S3, SWift..)는 폴더라는 개념이 없음
  - bucket을 사용함 (사용자/내문서/사진폴더 -> 진짜 폴더가 아니라 단지 문자열로 나열한 형태)
  - 트리 구조 메타데이터에 대한 오버헤드를 줄일 수 있음


# Erasure Coding
- 기본적으로 Ceph 풀은 모든 객체를 여러 디스크에 통째로 복사 함 (오버헤드가 큼)
- Erasure Coding의 경우 데이터를 데이터 block과 parity block으로 나눠 저장함
<img width="644" height="422" alt="image" src="https://github.com/user-attachments/assets/b2e969a5-fc17-4f97-b2f3-ccc2e530f8cf" />

- OSD (object storage daemon): 물리적인 디스크를 관리하는 프로그램 (각 디스크마다 한개씩 붙음)
- K: 데이터 청크 수, M은 OSD의 수
- Replication: 원본을 똑같이 3개의 OSD(디스크)에 복사함
  - 디스크 2개가 동시에 고장 나도 1개가 살아있으니 안전하지만, 저장 용량을 3배(3TB)나 차지합니다.
- Erasure Coding: 데이터를 2개(K1, K2)로 쪼개고, 복구를 위한 수학적 패리티를 2개(M1, M2) 만들어 총 4개의 OSD에 나눔.
  - 여기서도 디스크 2개(m=2)가 고장 나도 데이터를 복원할 수 있는데, 저장 용량은 딱 2배(2TB)만 차지함

# RGW 계층 구조
<img width="627" height="324" alt="image" src="https://github.com/user-attachments/assets/1217c744-2bdd-4fa1-818a-ff0c07da999b" />

- realm은 하나 이상의 클러스터에 걸치고, realm은 하나 이상의 zone group을 가짐
- 각 zone group 안에 여러 zone이 있고, zone안에 object store가 위치함
- master zone group의 master zone이 메타데이터 마스터 존으로 지정됨
  - 사용자, 버킷, 메타데이터 등 모든 변경은 이 마스터 존에 먼저 쓰여지고 다른 존으로 복제됨 (메타데이터는 한 곳에 먼저 쓰고, 데이터는 아무 데나 쓰고 양방향 복제)
  - 이 방식은 객체 데이터가 어느 존에나 쓰여지고 양방향 복제되는 데이터 동기화와 차별점임

### Operator 패턴
- Rook 오퍼레이터는 사용자가 CephObjectStore 단일 CR만 선언하면 내부적으로 Default 토폴로지를 생성하여 단일 클러스터 내에 격리된 Pod를 프로비저닝함
- 글로벌 동기화가 필요할 때는 사용자가 4개의 CR(Realm, ZoneGroup, Zone, Store)을 종속성에 맞게 선언하여, 오퍼레이터가 멀티사이트 동기화가 가능한 확장된 형태의 RGW 인프라를 프로비저닝하도록 해야함
- 즉, 클러스터 A와 클러스터B를 동기화할려면 global topology 선언이 필요함
  - 4개의 YAML 파일을 작성하여 4개의 개별 CR을 API 서버에 배포합니다.
  1. CephObjectRealm (CR) 생성
  2. CephObjectZoneGroup (CR) 생성 ➡️ spec.realm 필드에 앞서 만든 Realm의 이름을 적어 Binding함
  3. CephObjectZone (CR) 생성 ➡️ spec.zoneGroup 필드에 상위 그룹을 참조함
  3. CephObjectStore (CR) 생성 ➡️ spec.zone 필드에 자신이 속할 Zone을 참조함
  - 오퍼레이터의 동작 (Reconciliation): Rook 오퍼레이터는 이 4개의 CR이 서로 체인처럼 Dependency를 맺고 있는 것을 확인함
  - 오퍼레이터는 이 명세에 따라 단순한 격리형 Pod가 아니라, 외부 클러스터의 Zone과 통신할 수 있도록 설정(ConfigMap)이 주입되고 외부 접근(Ingress/Service)이 열린 RGW Pod를 프로비저닝함

# 버킷 생성
<img width="637" height="392" alt="image" src="https://github.com/user-attachments/assets/ac79ed02-7cf3-4ee8-abed-4260ef1160aa" />

- 버킷은 스토리지 클래스를 정의하는 방식으로 생성하는데, 블록과 파일 스토리지에서 쓰는 패턴과 유사함
- 스토리지 클래스를 기반으로 클라이언트가 OBC를 생성해 버킷을 요청하고, OBC가 생성되면 Rook 버킷 프로비저너가 새 버킷을 생성함 
- OBC와 같은 이름·같은 네임스페이스에 Secret과 ConfigMap이 생성되는데, Secret은 애플리케이션 파드가 버킷에 접근하는 자격증명을 담고, ConfigMap은 버킷 엔드포인트 정보를 담아 파드가 사용함
- ceph 다중 스토어를 만들 때 격리는 RADOS 네임스페이스로 해결함
- 공유 Ceph 풀을 만들면 스토어마다 풀을 추가하는 오버헤드를 줄이고, 데이터 격리는 RADOS 네임스페이스로 강제되어 스토어 간 데이터 접근이 차단됨


