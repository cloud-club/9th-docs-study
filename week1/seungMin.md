# Rook(k8s, ceph) Documentation 개인 학습 커리큘럼

## 👤 학습자
전승민
## 📚 학습 자료
- Main: Rook 공식 Documentation (v1.9)
  - https://rook.github.io/docs/rook/latest-release/Getting-Started/intro/

참고 자료)
- Rook Design Docs (GitHub) & Kubernetes 공식 Docs
- Ceph 공식 Documentation


## 📖 전체 문서 목차
1. Rook Over View & Kubernetes Control Plane, Operator
2. Ceph Core Engine (RADOS): 분산 저장소의 해싱(CRUSH) 및 데이터 배치 알고리즘
3. Block Storage Provisioning Internals: CRD가 실제 스토리지 데몬(OSD)으로 변환되는 구조
4. Container Storage Interface (CSI): Kubelet과 스토리지 간의 IPC(gRPC) 통신 원리
5. Object Storage & Data Durability: RGW 아키텍처와 Erasure Coding
6. Self-Healing & Consensus
7. Storage Engine: On-disk 포맷(BlueStore) 및 통합 시스템 아키텍처


## 📅 주차별 커리큘럼

### 1주차 (OT)
- 스터디 진행 방식 소개 및 커리큘럼 공유

### 2주차
- K8s Operator Pattern:
  - https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
- K8s Custom Resources (CRD):
  - https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/


### 3주차
마스터 노드 없이 데이터를 분산 저장하는 CRUSH 알고리즘의 결정론적 라우팅 원리 파악
- Ceph Architecture Overview:
  - https://docs.ceph.com/en/latest/architecture/
- CRUSH Algorithm Internals:
  - https://docs.ceph.com/en/latest/rados/operations/crush-map/
- Monitor (MON) Config & Quorum:
  - https://docs.ceph.com/en/latest/rados/configuration/mon-config-ref/
- Placement Groups:
  - https://docs.ceph.com/en/latest/rados/operations/placement-groups/


### 4주차
- Rook CRD가 내부적으로 Linux 블록 디바이스(RBD)로 어떻게 매핑되고 프로비저닝되는지 추적
  - Ceph RBD Architecture:
- https://docs.ceph.com/en/latest/rbd/rbd-architecture/
  - Rook CephCluster CRD (latest):
- https://rook.io/docs/rook/latest-release/CRDs/Cluster/ceph-cluster-crd/
  - Rook OSD Pod Design (design/ 폴더, OSD 프로비저닝 설계):
- https://github.com/rook/rook/blob/master/design/ceph/dedicated-osd-pod.md
  - Rook ceph-volume 프로비저닝 설계 (참고):
- https://github.com/rook/rook/blob/master/design/ceph/ceph-volume-provisioning.md



### 5주차
K8s Kubelet과 Ceph 스토리지 간의 gRPC 통신 규격 및 마운트 네임스페이스 원리 학습
- K8s CSI Overview:
  - https://kubernetes.io/docs/concepts/storage/volumes/#csi
- K8s PersistentVolumes:
  - https://kubernetes.io/docs/concepts/storage/persistent-volumes/
- CSI Specification (GitHub):
  - https://github.com/container-storage-interface/spec/blob/master/spec.md
- Rook CSI Driver Design (design/ 폴더):
  - https://github.com/rook/rook/blob/master/design/ceph/ceph-csi-driver.md



### 6주차
S3 API가 내부 객체로 변환되는 과정(RGW)과 Erasure Coding의 패리티 연산 원리 이해
- Ceph RADOS Gateway (RGW) Internals:
  - https://docs.ceph.com/en/latest/radosgw/
- Ceph Erasure Coding Operations:
  - https://docs.ceph.com/en/latest/rados/operations/erasure-code/
- Rook Object Store (latest):
  - https://rook.io/docs/rook/latest-release/Storage-Configuration/Object-Storage-RGW/object-storage/
- Rook Object Store Design (design/ 폴더):
  - https://github.com/rook/rook/blob/master/design/ceph/object/store.md


### 7주차
분산 시스템에서 노드 장애를 감지하는 방법과 데이터 재분배의 내부 알고리즘 분석
- Ceph OSD Peering & Heartbeats:
  - https://docs.ceph.com/en/latest/dev/peering/
- Ceph Health Log 분석:
  - https://docs.ceph.com/en/latest/rados/troubleshooting/log-and-debug/
- Rook Disaster Recovery (latest):
  - https://rook.io/docs/rook/latest-release/Troubleshooting/disaster-recovery/
- Rook Upgrade Design (rolling upgrade 원리):
  - https://github.com/rook/rook/blob/master/design/ceph/upgrade.md


### 8주차
BlueStore 아키텍처
- Ceph BlueStore Config & Internals:
  - https://docs.ceph.com/en/latest/rados/configuration/bluestore-config-ref/
- Rook Stretch Cluster Design (design/ 폴더):
  - https://github.com/rook/rook/blob/master/design/ceph/ceph-stretch-cluster.md
- Rook StorageClassDeviceSet Design (심화, OSD-PVC 연동 원리):
  - https://github.com/rook/rook/blob/master/design/ceph/storage-class-device-set.md
