# Rook
- 클라우드 네이티브 환경에서 스토리지를 관리하기 위한 오픈소스 클라우드 네이티브 스토리지 오케스트레이터입니다.
- Rook은 기본적으로 Kubernetes의 확장 기능인 Operator 패턴을 사용하여 스토리지 시스템을 배포, 업그레이드, 확장, 복구 및 모니터링 합니다.
- 이는 사용자가 복잡한 스토리지 시스템을 yaml 파일을 통해 선언적인 방식으로 관리할 수 있게 해주며, Kubernetes의 방식으로 스토리지 리소스를 다룰 수 있습니다.

# Rook Ceph 클러스터
- Rook Ceph 클러스터는 Rook을 사용하여 Kubernetes 환경에 배포된 Ceph 스토리지 클러스터를 의미합니다.
- Rook은 Ceph 클러스터의 배포, 구성, 확장, 업그레이드 및 모니터링을 자동화하여 Kubernetes 관리자가 복잡한 Ceph 운영 지식 없이도 안정적인 분산 스토리지 환경을 구축할 수 있게 해줍니다.

## Rook Ceph 클러스터 구성요소
- Ceph 모니터(MON): 클러스터 상태를 유지하고 모니터링하는 데몬
- Ceph 관리자(MGR): 클러스터 상태 정보를 수집하고 대시보드를 제공하는 데몬
- Ceph OSD(Object Storage Daemon): 실제 데이터를 저장하고 처리하는 데몬
- Ceph MDS(Metadata Server): CephFS를 위한 메타데이터 서비스를 제공하는 데몬(파일 시스템 사용 시)
- Rook Operator: Ceph 클러스터를 관리하는 Kubernetes Operator

## 스토리지 활용 방식
- Rook Ceph 클러스터는 Kubernetes 워커 노드의 물리적 스토리지 자원을 활용하여 분산 스토리지 시스템을 구축합니다.
- Kubernetes 클러스터에서 추가적인 전용 스토리지 시스템 없이 Ceph 기반의 고가용성 스토리지를 구축할 수 있습니다.
1. 블록 스토리지(Raw 디바이스)활용
  - 워커 노드에 연결된 마운트되지 않은 디스크 또는 파티션이 없는 디스크를 Ceph OSD의 데이터 저장소로 사용합니다.
  - Ceph OSD는 ceph-volume을 이용해 자동으로 디스크를 포맷하고 관리합니다.
  - 별도의 파일 시스템을 필요로 하지 않으며, Ceph의 분산 데이터 저장 방식에 최적화되어 있습니다.
2. PVC 기반 볼륨 제공
  - PVC를 요청하면 Rook Ceph가 Ceph RBD(블록 스토리지) 또는 CephFS(파일 스토리지)를 생성하여 제공합니다.
  - Rook Ceph는 K8s의 PV를 제공하는 CSI 드라이버 역할을 합니다.

## Rook Ceph 클러스터 구성
Rook Operator를 k8s에 배포하면 해당 operator가 k8s API Server에 CRD를 등록하고 사용하는 구조입니다.
- K8s의 Rook Operator pod가 수행 되면서 k8s API Server를 watch 합니다.
- 사용자 요청(CR)이 들어오면 대기중인 operator가 작업을 시작합니다.
- CRD (Custom Resource Definition): CephCluster같은 스토리지 용어를  사용하기 위해 추가하는 명세서를 말합니다.
```
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: cephclusters.ceph.rook.io   # K8s에 등록될 자원의 전체 이름
spec:
  group: ceph.rook.io               # API 그룹 (주소 같은 역할)
  names:
    kind: CephCluster               # 우리가 실제로 부를 이름
    plural: cephclusters            # 복수형 이름
  scope: Namespaced                 # 특정 네임스페이스 안에서만 살 것인지 여부
  versions:
    - name: v1
      schema:
        openAPIV3Schema:            # ★ 핵심: 이 자원이 가져야 할 '데이터 규격(타입)'을 정의
          type: object
          properties:
            spec:
              type: object
              properties:
                cephVersion:
                  type: object      # cephVersion 안에는 image라는 문자열이 와야 한다고 정의
                  properties:
                    image:
                      type: string
```

## Rook operator가 ceph 클러스터 동적으로 업데이트하는 방법
```
사용자가 CR의 디스크 목록 수정
       ↓
Operator가 Watch로 감지 (신규/이전 오브젝트 둘 다 받음)
       ↓
Diff 계산 (K8S API 서버를 감시하던 Rook Operator가 변화를 감지함)
       ↓
식별자 검증 (UUID Check)
       ↓
노드 단위로 하나씩 처리
```
- UUID로 체크하는 이유
  - 리눅스 재부팅으로 인한 단순한 마운트 이름(sdb, sda) 변경인지, 사용자가 진짜 디스크를 제거하는 것인지를 검증하기 위해서 입니다.

- 노드 단위로 순차적 진행
  - 디스크를 추가할 때
  - 새 OSD 파드를 생성하고 디스크를 포맷합니다.
  - 데이터 분배 지도(CRUSH Map)에 새 디스크를 등록합니다.
  - 기존 디스크들에 있던 데이터 일부가 새 디스크로 분산 배치됩니다.

  - 디스크를 제거할 때
    - 가중치 회수 (Reweight 0.0): 삭제할 디스크의 데이터 할당 가중치를 0으로 만듭니다.
    - Migration: 가중치가 0이 되었으므로, 시스템은 해당 디스크 안의 데이터를 다른 디스크들로 모두 빼냅니다.
    - 완료 대기: 모든 데이터가 안전하게 다른 곳에 복제될 때까지 프로세스를 끄지 않고 대기합니다.
    - 연결 해제 및 삭제: 데이터가 100% 빠져나간 빈 껍데기 상태가 확인되면, 그제야 CRUSH Map에서 디스크를 지우고 OSD 파드를 삭제합니다.



